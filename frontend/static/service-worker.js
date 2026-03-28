/**
 * Hearth Service Worker
 * Provides offline support, caching strategies, and background sync
 * 
 * Cache Strategies:
 * - Static assets: Cache-first with fallback to network
 * - API responses: Network-first with fallback to cache
 * - Message queue: IndexedDB for pending actions
 */

const CACHE_VERSION = 'v1';
const STATIC_CACHE = `hearth-static-${CACHE_VERSION}`;
const API_CACHE = `hearth-api-${CACHE_VERSION}`;
const IMAGE_CACHE = `hearth-images-${CACHE_VERSION}`;

// Static assets to pre-cache
const STATIC_ASSETS = [
  '/',
  '/favicon.ico',
  '/favicon.png',
  '/favicon-16.png',
  '/favicon-32.png',
  '/apple-touch-icon.png',
  '/hearth-logo.svg'
];

// API endpoints to cache (for offline viewing)
const CACHEABLE_API_PATTERNS = [
  /^\/api\/v1\/users\/me$/,
  /^\/api\/v1\/servers$/,
  /^\/api\/v1\/servers\/[^/]+$/,
  /^\/api\/v1\/channels\/[^/]+$/,
  /^\/api\/v1\/channels\/[^/]+\/messages/
];

// API endpoints that should use background sync
const SYNC_API_PATTERNS = [
  { pattern: /^\/api\/v1\/channels\/[^/]+\/messages$/, method: 'POST', tag: 'message-sync' },
  { pattern: /^\/api\/v1\/channels\/[^/]+\/typing$/, method: 'POST', tag: 'typing-sync' },
  { pattern: /^\/api\/v1\/messages\/[^/]+\/reactions/, method: 'POST', tag: 'reaction-sync' },
  { pattern: /^\/api\/v1\/messages\/[^/]+\/reactions/, method: 'DELETE', tag: 'reaction-sync' }
];

// IndexedDB for sync queue
const DB_NAME = 'hearth-sync';
const DB_VERSION = 1;
const SYNC_STORE = 'sync-queue';

/**
 * Open IndexedDB for sync queue
 */
function openSyncDB() {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);
    
    request.onerror = () => reject(request.error);
    request.onsuccess = () => resolve(request.result);
    
    request.onupgradeneeded = (event) => {
      const db = event.target.result;
      
      if (!db.objectStoreNames.contains(SYNC_STORE)) {
        const store = db.createObjectStore(SYNC_STORE, { keyPath: 'id' });
        store.createIndex('timestamp', 'timestamp', { unique: false });
        store.createIndex('tag', 'tag', { unique: false });
        store.createIndex('status', 'status', { unique: false });
      }
    };
  });
}

/**
 * Add request to sync queue
 */
async function addToSyncQueue(request, tag) {
  const db = await openSyncDB();
  const tx = db.transaction(SYNC_STORE, 'readwrite');
  const store = tx.objectStore(SYNC_STORE);
  
  const clonedRequest = request.clone();
  const body = await clonedRequest.text();
  
  const syncItem = {
    id: crypto.randomUUID(),
    url: request.url,
    method: request.method,
    headers: Object.fromEntries(request.headers.entries()),
    body,
    tag,
    timestamp: Date.now(),
    status: 'pending',
    retryCount: 0
  };
  
  await store.add(syncItem);
  db.close();
  
  // Register for background sync if available
  if ('sync' in self.registration) {
    try {
      await self.registration.sync.register(tag);
    } catch (e) {
      console.warn('[SW] Background sync registration failed:', e);
    }
  }
  
  return syncItem.id;
}

/**
 * Get pending items from sync queue
 */
async function getPendingSyncItems(tag = null) {
  const db = await openSyncDB();
  const tx = db.transaction(SYNC_STORE, 'readonly');
  const store = tx.objectStore(SYNC_STORE);
  
  return new Promise((resolve, reject) => {
    let request;
    if (tag) {
      const index = store.index('tag');
      request = index.getAll(tag);
    } else {
      const index = store.index('status');
      request = index.getAll('pending');
    }
    
    request.onsuccess = () => {
      db.close();
      resolve(request.result.filter(item => item.status === 'pending'));
    };
    request.onerror = () => {
      db.close();
      reject(request.error);
    };
  });
}

/**
 * Update sync item status
 */
async function updateSyncItem(id, updates) {
  const db = await openSyncDB();
  const tx = db.transaction(SYNC_STORE, 'readwrite');
  const store = tx.objectStore(SYNC_STORE);
  
  return new Promise((resolve, reject) => {
    const getRequest = store.get(id);
    
    getRequest.onsuccess = () => {
      const item = getRequest.result;
      if (item) {
        const updated = { ...item, ...updates };
        const putRequest = store.put(updated);
        putRequest.onsuccess = () => {
          db.close();
          resolve(updated);
        };
        putRequest.onerror = () => {
          db.close();
          reject(putRequest.error);
        };
      } else {
        db.close();
        resolve(null);
      }
    };
    getRequest.onerror = () => {
      db.close();
      reject(getRequest.error);
    };
  });
}

/**
 * Remove item from sync queue
 */
async function removeSyncItem(id) {
  const db = await openSyncDB();
  const tx = db.transaction(SYNC_STORE, 'readwrite');
  const store = tx.objectStore(SYNC_STORE);
  
  return new Promise((resolve, reject) => {
    const request = store.delete(id);
    request.onsuccess = () => {
      db.close();
      resolve();
    };
    request.onerror = () => {
      db.close();
      reject(request.error);
    };
  });
}

/**
 * Process sync queue items
 */
async function processSyncQueue(tag = null) {
  const items = await getPendingSyncItems(tag);
  const results = [];
  
  for (const item of items) {
    try {
      const response = await fetch(item.url, {
        method: item.method,
        headers: item.headers,
        body: item.body || undefined
      });
      
      if (response.ok) {
        await removeSyncItem(item.id);
        results.push({ id: item.id, success: true });
        
        // Notify clients of successful sync
        const clients = await self.clients.matchAll();
        clients.forEach(client => {
          client.postMessage({
            type: 'SYNC_COMPLETE',
            id: item.id,
            tag: item.tag
          });
        });
      } else if (response.status >= 400 && response.status < 500) {
        // Client error - don't retry
        await updateSyncItem(item.id, { status: 'failed', error: `HTTP ${response.status}` });
        results.push({ id: item.id, success: false, error: `HTTP ${response.status}` });
      } else {
        // Server error - mark for retry
        await updateSyncItem(item.id, { 
          retryCount: item.retryCount + 1,
          lastError: `HTTP ${response.status}`
        });
        results.push({ id: item.id, success: false, retry: true });
      }
    } catch (error) {
      // Network error - keep in queue for retry
      await updateSyncItem(item.id, { 
        retryCount: item.retryCount + 1,
        lastError: error.message
      });
      results.push({ id: item.id, success: false, retry: true, error: error.message });
    }
  }
  
  return results;
}

/**
 * Check if URL matches any cacheable pattern
 */
function isCacheableAPI(url) {
  const path = new URL(url).pathname;
  return CACHEABLE_API_PATTERNS.some(pattern => pattern.test(path));
}

/**
 * Check if request should use background sync
 */
function getSyncTag(request) {
  const path = new URL(request.url).pathname;
  const match = SYNC_API_PATTERNS.find(
    item => item.pattern.test(path) && item.method === request.method
  );
  return match ? match.tag : null;
}

/**
 * Cache-first strategy for static assets
 */
async function cacheFirst(request, cacheName) {
  const cached = await caches.match(request);
  if (cached) {
    return cached;
  }
  
  try {
    const response = await fetch(request);
    if (response.ok) {
      const cache = await caches.open(cacheName);
      cache.put(request, response.clone());
    }
    return response;
  } catch (error) {
    console.error('[SW] Cache-first fetch failed:', error);
    throw error;
  }
}

/**
 * Network-first strategy for API calls
 */
async function networkFirst(request, cacheName) {
  try {
    const response = await fetch(request);
    if (response.ok && request.method === 'GET') {
      const cache = await caches.open(cacheName);
      cache.put(request, response.clone());
    }
    return response;
  } catch (error) {
    const cached = await caches.match(request);
    if (cached) {
      // Add header to indicate cached response
      const headers = new Headers(cached.headers);
      headers.set('X-Hearth-Cached', 'true');
      return new Response(cached.body, {
        status: cached.status,
        statusText: cached.statusText,
        headers
      });
    }
    throw error;
  }
}

/**
 * Stale-while-revalidate for images
 */
async function staleWhileRevalidate(request, cacheName) {
  const cache = await caches.open(cacheName);
  const cached = await cache.match(request);
  
  const fetchPromise = fetch(request).then(response => {
    if (response.ok) {
      cache.put(request, response.clone());
    }
    return response;
  }).catch(() => null);
  
  return cached || fetchPromise;
}

// Install event - pre-cache static assets
self.addEventListener('install', (event) => {
  console.log('[SW] Installing service worker');
  
  event.waitUntil(
    caches.open(STATIC_CACHE)
      .then(cache => cache.addAll(STATIC_ASSETS))
      .then(() => self.skipWaiting())
  );
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
  console.log('[SW] Activating service worker');
  
  event.waitUntil(
    caches.keys()
      .then(keys => Promise.all(
        keys
          .filter(key => key.startsWith('hearth-') && 
                        !key.endsWith(`-${CACHE_VERSION}`))
          .map(key => caches.delete(key))
      ))
      .then(() => self.clients.claim())
  );
});

// Fetch event - handle requests with appropriate strategy
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);
  
  // Skip non-GET requests for caching, but handle sync
  if (request.method !== 'GET') {
    const syncTag = getSyncTag(request);
    
    if (syncTag) {
      event.respondWith(
        fetch(request.clone()).catch(async (error) => {
          // Network failed - queue for background sync
          const syncId = await addToSyncQueue(request, syncTag);
          
          // Return a synthetic response indicating queued
          return new Response(JSON.stringify({
            queued: true,
            syncId,
            message: 'Request queued for sync'
          }), {
            status: 202,
            headers: {
              'Content-Type': 'application/json',
              'X-Hearth-Queued': 'true'
            }
          });
        })
      );
    }
    return;
  }
  
  // Skip cross-origin requests (except for specific CDNs)
  if (url.origin !== location.origin) {
    return;
  }
  
  // API requests - network first
  if (url.pathname.startsWith('/api/')) {
    if (isCacheableAPI(url.href)) {
      event.respondWith(networkFirst(request, API_CACHE));
    }
    return;
  }
  
  // Images - stale while revalidate
  if (request.destination === 'image' || 
      url.pathname.match(/\.(png|jpg|jpeg|gif|webp|svg|ico)$/)) {
    event.respondWith(staleWhileRevalidate(request, IMAGE_CACHE));
    return;
  }
  
  // Static assets - cache first
  event.respondWith(cacheFirst(request, STATIC_CACHE));
});

// Background sync event
self.addEventListener('sync', (event) => {
  console.log('[SW] Background sync triggered:', event.tag);
  
  event.waitUntil(processSyncQueue(event.tag));
});

// Periodic background sync (if supported)
self.addEventListener('periodicsync', (event) => {
  if (event.tag === 'hearth-sync') {
    event.waitUntil(processSyncQueue());
  }
});

// Message event - handle commands from main thread
self.addEventListener('message', async (event) => {
  const { type, data } = event.data || {};
  
  switch (type) {
    case 'SKIP_WAITING':
      self.skipWaiting();
      break;
      
    case 'GET_SYNC_QUEUE':
      const items = await getPendingSyncItems(data?.tag);
      event.source.postMessage({
        type: 'SYNC_QUEUE',
        items
      });
      break;
      
    case 'PROCESS_SYNC':
      const results = await processSyncQueue(data?.tag);
      event.source.postMessage({
        type: 'SYNC_RESULTS',
        results
      });
      break;
      
    case 'CLEAR_CACHE':
      const cacheNames = await caches.keys();
      await Promise.all(
        cacheNames
          .filter(name => name.startsWith('hearth-'))
          .map(name => caches.delete(name))
      );
      event.source.postMessage({ type: 'CACHE_CLEARED' });
      break;
      
    case 'CACHE_URLS':
      if (data?.urls) {
        const cache = await caches.open(STATIC_CACHE);
        await cache.addAll(data.urls);
        event.source.postMessage({ type: 'URLS_CACHED' });
      }
      break;
  }
});

// Push notification event
self.addEventListener('push', (event) => {
  if (!event.data) return;
  
  try {
    const data = event.data.json();
    
    const options = {
      body: data.body || '',
      icon: '/favicon-32.png',
      badge: '/favicon-16.png',
      tag: data.tag || 'hearth-notification',
      data: data.data || {},
      vibrate: [100, 50, 100],
      actions: data.actions || []
    };
    
    event.waitUntil(
      self.registration.showNotification(data.title || 'Hearth', options)
    );
  } catch (error) {
    console.error('[SW] Push notification error:', error);
  }
});

// Notification click event
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  
  const data = event.notification.data || {};
  const urlToOpen = data.url || '/';
  
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true })
      .then(clients => {
        // Check if there's already a window open
        for (const client of clients) {
          if (client.url.includes(urlToOpen) && 'focus' in client) {
            return client.focus();
          }
        }
        // Open a new window
        if (self.clients.openWindow) {
          return self.clients.openWindow(urlToOpen);
        }
      })
  );
});

console.log('[SW] Service worker loaded');
