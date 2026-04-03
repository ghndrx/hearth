package co.hndrx.hearth.core.storage

import androidx.room.Dao
import androidx.room.Database
import androidx.room.Entity
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.PrimaryKey
import androidx.room.Query
import androidx.room.RoomDatabase
import kotlinx.coroutines.flow.Flow

// ---------- Entities ----------

@Entity(tableName = "cached_servers")
data class CachedServer(
    @PrimaryKey val id: String,
    val name: String,
    val iconUrl: String?,
    val bannerUrl: String?,
    val description: String?,
    val ownerId: String,
    val createdAt: String?,
    val lastUpdated: Long = System.currentTimeMillis(),
)

@Entity(tableName = "cached_channels")
data class CachedChannel(
    @PrimaryKey val id: String,
    val serverId: String?,
    val name: String,
    val type: String,
    val topic: String?,
    val position: Int,
    val nsfw: Boolean,
    val parentId: String?,
    val lastMessageId: String?,
    val createdAt: String?,
    val lastUpdated: Long = System.currentTimeMillis(),
)

@Entity(tableName = "cached_messages")
data class CachedMessage(
    @PrimaryKey val id: String,
    val channelId: String,
    val authorId: String,
    val authorUsername: String?,
    val authorAvatarUrl: String?,
    val content: String,
    val type: Int,
    val replyTo: String?,
    val pinned: Boolean,
    val createdAt: String,
    val editedAt: String?,
    val lastUpdated: Long = System.currentTimeMillis(),
)

// ---------- DAOs ----------

@Dao
interface ServerDao {
    @Query("SELECT * FROM cached_servers ORDER BY name ASC")
    fun getAllServers(): Flow<List<CachedServer>>

    @Query("SELECT * FROM cached_servers WHERE id = :serverId")
    suspend fun getServerById(serverId: String): CachedServer?

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertServers(servers: List<CachedServer>)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertServer(server: CachedServer)

    @Query("DELETE FROM cached_servers")
    suspend fun deleteAll()
}

@Dao
interface ChannelDao {
    @Query("SELECT * FROM cached_channels WHERE serverId = :serverId ORDER BY position ASC")
    fun getChannelsForServer(serverId: String): Flow<List<CachedChannel>>

    @Query("SELECT * FROM cached_channels WHERE id = :channelId")
    suspend fun getChannelById(channelId: String): CachedChannel?

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertChannels(channels: List<CachedChannel>)

    @Query("DELETE FROM cached_channels WHERE serverId = :serverId")
    suspend fun deleteChannelsForServer(serverId: String)

    @Query("DELETE FROM cached_channels")
    suspend fun deleteAll()
}

@Dao
interface MessageDao {
    @Query("SELECT * FROM cached_messages WHERE channelId = :channelId ORDER BY createdAt DESC LIMIT :limit")
    fun getMessagesForChannel(channelId: String, limit: Int = 50): Flow<List<CachedMessage>>

    @Query("SELECT * FROM cached_messages WHERE id = :messageId")
    suspend fun getMessageById(messageId: String): CachedMessage?

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertMessages(messages: List<CachedMessage>)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertMessage(message: CachedMessage)

    @Query("DELETE FROM cached_messages WHERE channelId = :channelId")
    suspend fun deleteMessagesForChannel(channelId: String)

    @Query("DELETE FROM cached_messages")
    suspend fun deleteAll()
}

// ---------- Database ----------

@Database(
    entities = [CachedServer::class, CachedChannel::class, CachedMessage::class],
    version = 1,
    exportSchema = false,
)
abstract class HearthDatabase : RoomDatabase() {
    abstract fun serverDao(): ServerDao
    abstract fun channelDao(): ChannelDao
    abstract fun messageDao(): MessageDao
}
