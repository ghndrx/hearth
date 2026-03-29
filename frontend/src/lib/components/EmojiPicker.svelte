<script lang="ts">
  import { createEventDispatcher, onMount, onDestroy, tick } from 'svelte';
  import { handleListKeyboard } from '$lib/utils/keyboard';

  export let show = false;

  const dispatch = createEventDispatcher<{ select: string; close: void }>();

  // Grid columns for keyboard navigation
  const GRID_COLUMNS = 9;
  let focusedEmojiIndex = -1;
  let emojisContainer: HTMLDivElement;

  // Skin tone modifiers
  const skinTones = [
    { name: 'Default', modifier: '', color: '#ffcc4d' },
    { name: 'Light', modifier: '\u{1F3FB}', color: '#f7dece' },
    { name: 'Medium-Light', modifier: '\u{1F3FC}', color: '#e0bb95' },
    { name: 'Medium', modifier: '\u{1F3FD}', color: '#bf8f68' },
    { name: 'Medium-Dark', modifier: '\u{1F3FE}', color: '#9b643d' },
    { name: 'Dark', modifier: '\u{1F3FF}', color: '#594539' }
  ];

  // Emojis that support skin tone modifiers
  const skinToneEmojis = new Set([
    '👋', '🤚', '🖐️', '✋', '🖖', '👌', '🤌', '🤏', '✌️', '🤞', '🤟', '🤘', '🤙',
    '👈', '👉', '👆', '🖕', '👇', '☝️', '👍', '👎', '✊', '👊', '🤛', '🤜', '👏',
    '🙌', '👐', '🤲', '🤝', '🙏', '✍️', '💪', '🦵', '🦶', '👂', '🦻', '👃', '👶',
    '👧', '🧒', '👦', '👩', '🧑', '👨', '👩‍🦱', '🧑‍🦱', '👨‍🦱', '👩‍🦰', '🧑‍🦰', '👨‍🦰',
    '👱‍♀️', '👱', '👱‍♂️', '👩‍🦳', '🧑‍🦳', '👨‍🦳', '👩‍🦲', '🧑‍🦲', '👨‍🦲', '🧔', '👵', '🧓',
    '👴', '👲', '👳‍♀️', '👳', '👳‍♂️', '🧕', '👮‍♀️', '👮', '👮‍♂️', '👷‍♀️', '👷', '👷‍♂️',
    '💂‍♀️', '💂', '💂‍♂️', '🕵️‍♀️', '🕵️', '🕵️‍♂️', '👩‍⚕️', '🧑‍⚕️', '👨‍⚕️', '👩‍🌾', '🧑‍🌾',
    '👨‍🌾', '👩‍🍳', '🧑‍🍳', '👨‍🍳', '👩‍🎓', '🧑‍🎓', '👨‍🎓', '👩‍🎤', '🧑‍🎤', '👨‍🎤', '👩‍🏫',
    '🧑‍🏫', '👨‍🏫', '👩‍🏭', '🧑‍🏭', '👨‍🏭', '👩‍💻', '🧑‍💻', '👨‍💻', '👩‍💼', '🧑‍💼', '👨‍💼',
    '👩‍🔧', '🧑‍🔧', '👨‍🔧', '👩‍🔬', '🧑‍🔬', '👨‍🔬', '👩‍🎨', '🧑‍🎨', '👨‍🎨', '👩‍🚒', '🧑‍🚒',
    '👨‍🚒', '👩‍✈️', '🧑‍✈️', '👨‍✈️', '👩‍🚀', '🧑‍🚀', '👨‍🚀', '👩‍⚖️', '🧑‍⚖️', '👨‍⚖️', '👰‍♀️',
    '👰', '👰‍♂️', '🤵‍♀️', '🤵', '🤵‍♂️', '👸', '🤴', '🥷', '🦸‍♀️', '🦸', '🦸‍♂️', '🦹‍♀️',
    '🦹', '🦹‍♂️', '🤶', '🎅', '🧙‍♀️', '🧙', '🧙‍♂️', '🧝‍♀️', '🧝', '🧝‍♂️', '🧛‍♀️', '🧛',
    '🧛‍♂️', '🧟‍♀️', '🧟', '🧟‍♂️', '🧞‍♀️', '🧞', '🧞‍♂️', '🧜‍♀️', '🧜', '🧜‍♂️', '🧚‍♀️', '🧚',
    '🧚‍♂️', '👼', '🤰', '🤱', '👩‍🍼', '🧑‍🍼', '👨‍🍼', '🙇‍♀️', '🙇', '🙇‍♂️', '💁‍♀️', '💁',
    '💁‍♂️', '🙅‍♀️', '🙅', '🙅‍♂️', '🙆‍♀️', '🙆', '🙆‍♂️', '🙋‍♀️', '🙋', '🙋‍♂️', '🧏‍♀️', '🧏',
    '🧏‍♂️', '🤦‍♀️', '🤦', '🤦‍♂️', '🤷‍♀️', '🤷', '🤷‍♂️', '🙎‍♀️', '🙎', '🙎‍♂️', '🙍‍♀️', '🙍',
    '🙍‍♂️', '💇‍♀️', '💇', '💇‍♂️', '💆‍♀️', '💆', '💆‍♂️', '🧖‍♀️', '🧖', '🧖‍♂️', '💅', '🤳',
    '💃', '🕺', '🕴️', '👩‍🦽', '🧑‍🦽', '👨‍🦽', '👩‍🦼', '🧑‍🦼', '👨‍🦼', '🚶‍♀️', '🚶', '🚶‍♂️',
    '👩‍🦯', '🧑‍🦯', '👨‍🦯', '🧎‍♀️', '🧎', '🧎‍♂️', '🏃‍♀️', '🏃', '🏃‍♂️', '🧍‍♀️', '🧍', '🧍‍♂️',
    '🏋️‍♀️', '🏋️', '🏋️‍♂️', '🤸‍♀️', '🤸', '🤸‍♂️', '⛹️‍♀️', '⛹️', '⛹️‍♂️', '🤾‍♀️', '🤾', '🤾‍♂️',
    '🏌️‍♀️', '🏌️', '🏌️‍♂️', '🏇', '🧘‍♀️', '🧘', '🧘‍♂️', '🏄‍♀️', '🏄', '🏄‍♂️', '🏊‍♀️', '🏊',
    '🏊‍♂️', '🤽‍♀️', '🤽', '🤽‍♂️', '🚣‍♀️', '🚣', '🚣‍♂️', '🧗‍♀️', '🧗', '🧗‍♂️', '🚵‍♀️', '🚵',
    '🚵‍♂️', '🚴‍♀️', '🚴', '🚴‍♂️', '🛀', '🛌'
  ]);

  // Emoji categories with Discord-like icons
  const categories = [
    {
      id: 'recent',
      name: 'Recently Used',
      icon: '🕒',
      emojis: [] as string[]
    },
    {
      id: 'smileys',
      name: 'Smileys & Emotion',
      icon: '😀',
      emojis: ['😀', '😃', '😄', '😁', '😆', '😅', '🤣', '😂', '🙂', '🙃', '😉', '😊', '😇', '🥰', '😍', '🤩', '😘', '😗', '☺️', '😚', '😙', '🥲', '😋', '😛', '😜', '🤪', '😝', '🤑', '🤗', '🤭', '🤫', '🤔', '🤐', '🤨', '😐', '😑', '😶', '😏', '😒', '🙄', '😬', '🤥', '😌', '😔', '😪', '🤤', '😴', '😷', '🤒', '🤕', '🤢', '🤮', '🤧', '🥵', '🥶', '🥴', '😵', '🤯', '🤠', '🥳', '🥸', '😎', '🤓', '🧐', '😕', '😟', '🙁', '☹️', '😮', '😯', '😲', '😳', '🥺', '😦', '😧', '😨', '😰', '😥', '😢', '😭', '😱', '😖', '😣', '😞', '😓', '😩', '😫', '🥱', '😤', '😡', '😠', '🤬', '😈', '👿', '💀', '☠️', '💩', '🤡', '👹', '👺', '👻', '👽', '👾', '🤖', '😺', '😸', '😹', '😻', '😼', '😽', '🙀', '😿', '😾']
    },
    {
      id: 'people',
      name: 'People & Body',
      icon: '👋',
      emojis: ['👋', '🤚', '🖐️', '✋', '🖖', '👌', '🤌', '🤏', '✌️', '🤞', '🤟', '🤘', '🤙', '👈', '👉', '👆', '🖕', '👇', '☝️', '👍', '👎', '✊', '👊', '🤛', '🤜', '👏', '🙌', '👐', '🤲', '🤝', '🙏', '✍️', '💅', '🤳', '💪', '🦾', '🦿', '🦵', '🦶', '👂', '🦻', '👃', '🧠', '🫀', '🫁', '🦷', '🦴', '👀', '👁️', '👅', '👄', '👶', '🧒', '👦', '👧', '🧑', '👱', '👨', '🧔', '👩', '🧓', '👴', '👵', '🙍', '🙎', '🙅', '🙆', '💁', '🙋', '🧏', '🙇', '🤦', '🤷', '👮', '🕵️', '💂', '🥷', '👷', '🤴', '👸', '👳', '👲', '🧕', '🤵', '👰', '🤰', '🤱', '👼', '🎅', '🤶', '🦸', '🦹', '🧙', '🧚', '🧛', '🧜', '🧝', '🧞', '🧟', '💆', '💇', '🚶', '🧍', '🧎', '🏃', '💃', '🕺', '🕴️', '👯', '🧖', '🧗', '🤸', '🏌️', '🏇', '⛷️', '🏂', '🏋️', '🤼', '🤽', '🤾', '🤺', '⛹️', '🏊', '🚣', '🧘', '🛀', '🛌', '👭', '👫', '👬', '💏', '💑', '👪', '👨‍👩‍👦', '👨‍👩‍👧', '👨‍👩‍👧‍👦', '👨‍👩‍👦‍👦', '👨‍👩‍👧‍👧', '👨‍👦', '👨‍👦‍👦', '👨‍👧', '👨‍👧‍👦', '👨‍👧‍👧', '👩‍👦', '👩‍👦‍👦', '👩‍👧', '👩‍👧‍👦', '👩‍👧‍👧', '🗣️', '👤', '👥', '🫂']
    },
    {
      id: 'nature',
      name: 'Animals & Nature',
      icon: '🐻',
      emojis: ['🐶', '🐱', '🐭', '🐹', '🐰', '🦊', '🐻', '🐼', '🐻‍❄️', '🐨', '🐯', '🦁', '🐮', '🐷', '🐽', '🐸', '🐵', '🙈', '🙉', '🙊', '🐒', '🐔', '🐧', '🐦', '🐤', '🐣', '🐥', '🦆', '🦅', '🦉', '🦇', '🐺', '🐗', '🐴', '🦄', '🐝', '🪱', '🐛', '🦋', '🐌', '🐞', '🐜', '🪰', '🪲', '🪳', '🦟', '🦗', '🕷️', '🕸️', '🦂', '🐢', '🐍', '🦎', '🦖', '🦕', '🐙', '🦑', '🦐', '🦞', '🦀', '🐡', '🐠', '🐟', '🐬', '🐳', '🐋', '🦈', '🐊', '🐅', '🐆', '🦓', '🦍', '🦧', '🦣', '🐘', '🦛', '🦏', '🐪', '🐫', '🦒', '🦘', '🦬', '🐃', '🐂', '🐄', '🐎', '🐖', '🐏', '🐑', '🦙', '🐐', '🦌', '🐕', '🐩', '🦮', '🐕‍🦺', '🐈', '🐈‍⬛', '🪶', '🐓', '🦃', '🦤', '🦚', '🦜', '🦢', '🦩', '🕊️', '🐇', '🦝', '🦨', '🦡', '🦫', '🦦', '🦥', '🐁', '🐀', '🐿️', '🦔', '🐾', '🐉', '🐲', '🌵', '🎄', '🌲', '🌳', '🌴', '🪵', '🌱', '🌿', '☘️', '🍀', '🎍', '🪴', '🎋', '🍃', '🍂', '🍁', '🍄', '🐚', '🪨', '🌾', '💐', '🌷', '🌹', '🥀', '🌺', '🌸', '🌼', '🌻', '🌞', '🌝', '🌛', '🌜', '🌚', '🌕', '🌖', '🌗', '🌘', '🌑', '🌒', '🌓', '🌔', '🌙', '🌎', '🌍', '🌏', '🪐', '💫', '⭐', '🌟', '✨', '⚡', '☄️', '💥', '🔥', '🌪️', '🌈', '☀️', '🌤️', '⛅', '🌥️', '☁️', '🌦️', '🌧️', '⛈️', '🌩️', '🌨️', '❄️', '☃️', '⛄', '🌬️', '💨', '💧', '💦', '☔', '☂️', '🌊', '🌫️']
    },
    {
      id: 'food',
      name: 'Food & Drink',
      icon: '🍔',
      emojis: ['🍇', '🍈', '🍉', '🍊', '🍋', '🍌', '🍍', '🥭', '🍎', '🍏', '🍐', '🍑', '🍒', '🍓', '🫐', '🥝', '🍅', '🫒', '🥥', '🥑', '🍆', '🥔', '🥕', '🌽', '🌶️', '🫑', '🥒', '🥬', '🥦', '🧄', '🧅', '🍄', '🥜', '🌰', '🍞', '🥐', '🥖', '🫓', '🥨', '🥯', '🥞', '🧇', '🧀', '🍖', '🍗', '🥩', '🥓', '🍔', '🍟', '🍕', '🌭', '🥪', '🌮', '🌯', '🫔', '🥙', '🧆', '🥚', '🍳', '🥘', '🍲', '🫕', '🥣', '🥗', '🍿', '🧈', '🧂', '🥫', '🍱', '🍘', '🍙', '🍚', '🍛', '🍜', '🍝', '🍠', '🍢', '🍣', '🍤', '🍥', '🥮', '🍡', '🥟', '🥠', '🥡', '🦀', '🦞', '🦐', '🦑', '🦪', '🍦', '🍧', '🍨', '🍩', '🍪', '🎂', '🍰', '🧁', '🥧', '🍫', '🍬', '🍭', '🍮', '🍯', '🍼', '🥛', '☕', '🫖', '🍵', '🍶', '🍾', '🍷', '🍸', '🍹', '🍺', '🍻', '🥂', '🥃', '🥤', '🧋', '🧃', '🧉', '🧊', '🥢', '🍽️', '🍴', '🥄', '🔪', '🏺']
    },
    {
      id: 'activities',
      name: 'Activities',
      icon: '⚽',
      emojis: ['⚽', '🏀', '🏈', '⚾', '🥎', '🎾', '🏐', '🏉', '🥏', '🎱', '🪀', '🏓', '🏸', '🏒', '🏑', '🥍', '🏏', '🪃', '🥅', '⛳', '🪁', '🏹', '🎣', '🤿', '🥊', '🥋', '🎽', '🛹', '🛼', '🛷', '⛸️', '🥌', '🎿', '⛷️', '🏂', '🪂', '🏋️', '🤼', '🤸', '🤺', '⛹️', '🤾', '🏌️', '🏇', '🧘', '🏄', '🏊', '🤽', '🚣', '🧗', '🚵', '🚴', '🏆', '🥇', '🥈', '🥉', '🏅', '🎖️', '🏵️', '🎗️', '🎫', '🎟️', '🎪', '🤹', '🎭', '🩰', '🎨', '🎬', '🎤', '🎧', '🎼', '🎹', '🥁', '🪘', '🎷', '🎺', '🪗', '🎸', '🪕', '🎻', '🎲', '♟️', '🎯', '🎳', '🎮', '🎰', '🧩']
    },
    {
      id: 'travel',
      name: 'Travel & Places',
      icon: '🚗',
      emojis: ['🚗', '🚕', '🚙', '🚌', '🚎', '🏎️', '🚓', '🚑', '🚒', '🚐', '🛻', '🚚', '🚛', '🚜', '🦯', '🦽', '🦼', '🛴', '🚲', '🛵', '🏍️', '🛺', '🚨', '🚔', '🚍', '🚘', '🚖', '🚡', '🚠', '🚟', '🚃', '🚋', '🚞', '🚝', '🚄', '🚅', '🚈', '🚂', '🚆', '🚇', '🚊', '🚉', '✈️', '🛫', '🛬', '🛩️', '💺', '🛰️', '🚀', '🛸', '🚁', '🛶', '⛵', '🚤', '🛥️', '🛳️', '⛴️', '🚢', '⚓', '🪝', '⛽', '🚧', '🚦', '🚥', '🚏', '🗺️', '🗿', '🗽', '🗼', '🏰', '🏯', '🏟️', '🎡', '🎢', '🎠', '⛲', '⛱️', '🏖️', '🏝️', '🏜️', '🌋', '⛰️', '🏔️', '🗻', '🏕️', '⛺', '🛖', '🏠', '🏡', '🏘️', '🏚️', '🏗️', '🏭', '🏢', '🏬', '🏣', '🏤', '🏥', '🏦', '🏨', '🏪', '🏫', '🏩', '💒', '🏛️', '⛪', '🕌', '🕍', '🛕', '🕋', '⛩️', '🛤️', '🛣️', '🗾', '🎑', '🏞️', '🌅', '🌄', '🌠', '🎇', '🎆', '🌇', '🌆', '🏙️', '🌃', '🌌', '🌉', '🌁']
    },
    {
      id: 'objects',
      name: 'Objects',
      icon: '💡',
      emojis: ['⌚', '📱', '📲', '💻', '⌨️', '🖥️', '🖨️', '🖱️', '🖲️', '🕹️', '🗜️', '💽', '💾', '💿', '📀', '📼', '📷', '📸', '📹', '🎥', '📽️', '🎞️', '📞', '☎️', '📟', '📠', '📺', '📻', '🎙️', '🎚️', '🎛️', '🧭', '⏱️', '⏲️', '⏰', '🕰️', '⌛', '⏳', '📡', '🔋', '🔌', '💡', '🔦', '🕯️', '🪔', '🧯', '🛢️', '💸', '💵', '💴', '💶', '💷', '🪙', '💰', '💳', '💎', '⚖️', '🪜', '🧰', '🪛', '🔧', '🔨', '⚒️', '🛠️', '⛏️', '🪚', '🔩', '⚙️', '🪤', '🧱', '⛓️', '🧲', '🔫', '💣', '🧨', '🪓', '🔪', '🗡️', '⚔️', '🛡️', '🚬', '⚰️', '🪦', '⚱️', '🏺', '🔮', '📿', '🧿', '💈', '⚗️', '🔭', '🔬', '🕳️', '🩹', '🩺', '💊', '💉', '🩸', '🧬', '🦠', '🧫', '🧪', '🌡️', '🧹', '🪠', '🧺', '🧻', '🚽', '🚰', '🚿', '🛁', '🛀', '🧼', '🪥', '🪒', '🧽', '🪣', '🧴', '🛎️', '🔑', '🗝️', '🚪', '🪑', '🛋️', '🛏️', '🛌', '🧸', '🪆', '🖼️', '🪞', '🪟', '🛍️', '🛒', '🎁', '🎈', '🎏', '🎀', '🪄', '🎊', '🎉', '🎎', '🏮', '🎐', '🧧', '✉️', '📩', '📨', '📧', '💌', '📥', '📤', '📦', '🏷️', '📪', '📫', '📬', '📭', '📮', '📯', '📜', '📃', '📄', '📑', '🧾', '📊', '📈', '📉', '🗒️', '🗓️', '📆', '📅', '🗑️', '📇', '🗃️', '🗳️', '🗄️', '📋', '📁', '📂', '🗂️', '🗞️', '📰', '📓', '📔', '📒', '📕', '📗', '📘', '📙', '📚', '📖', '🔖', '🧷', '🔗', '📎', '🖇️', '📐', '📏', '🧮', '📌', '📍', '✂️', '🖊️', '🖋️', '✒️', '🖌️', '🖍️', '📝', '✏️', '🔍', '🔎', '🔏', '🔐', '🔒', '🔓']
    },
    {
      id: 'symbols',
      name: 'Symbols',
      icon: '❤️',
      emojis: ['❤️', '🧡', '💛', '💚', '💙', '💜', '🖤', '🤍', '🤎', '💔', '❣️', '💕', '💞', '💓', '💗', '💖', '💘', '💝', '💟', '☮️', '✝️', '☪️', '🕉️', '☸️', '✡️', '🔯', '🕎', '☯️', '☦️', '🛐', '⛎', '♈', '♉', '♊', '♋', '♌', '♍', '♎', '♏', '♐', '♑', '♒', '♓', '🆔', '⚛️', '🉑', '☢️', '☣️', '📴', '📳', '🈶', '🈚', '🈸', '🈺', '🈷️', '✴️', '🆚', '💮', '🉐', '㊙️', '㊗️', '🈴', '🈵', '🈹', '🈲', '🅰️', '🅱️', '🆎', '🆑', '🅾️', '🆘', '❌', '⭕', '🛑', '⛔', '📛', '🚫', '💯', '💢', '♨️', '🚷', '🚯', '🚳', '🚱', '🔞', '📵', '🚭', '❗', '❕', '❓', '❔', '‼️', '⁉️', '🔅', '🔆', '〽️', '⚠️', '🚸', '🔱', '⚜️', '🔰', '♻️', '✅', '🈯', '💹', '❇️', '✳️', '❎', '🌐', '💠', 'Ⓜ️', '🌀', '💤', '🏧', '🚾', '♿', '🅿️', '🛗', '🈳', '🈂️', '🛂', '🛃', '🛄', '🛅', '🚹', '🚺', '🚼', '⚧️', '🚻', '🚮', '🎦', '📶', '🈁', '🔣', 'ℹ️', '🔤', '🔡', '🔠', '🆖', '🆗', '🆙', '🆒', '🆕', '🆓', '0️⃣', '1️⃣', '2️⃣', '3️⃣', '4️⃣', '5️⃣', '6️⃣', '7️⃣', '8️⃣', '9️⃣', '🔟', '🔢', '#️⃣', '*️⃣', '⏏️', '▶️', '⏸️', '⏯️', '⏹️', '⏺️', '⏭️', '⏮️', '⏩', '⏪', '⏫', '⏬', '◀️', '🔼', '🔽', '➡️', '⬅️', '⬆️', '⬇️', '↗️', '↘️', '↙️', '↖️', '↕️', '↔️', '↪️', '↩️', '⤴️', '⤵️', '🔀', '🔁', '🔂', '🔄', '🔃', '🎵', '🎶', '➕', '➖', '➗', '✖️', '🟰', '♾️', '💲', '💱', '™️', '©️', '®️', '👁️‍🗨️', '🔚', '🔙', '🔛', '🔝', '🔜', '〰️', '➰', '➿', '✔️', '☑️', '🔘', '🔴', '🟠', '🟡', '🟢', '🔵', '🟣', '⚫', '⚪', '🟤', '🔺', '🔻', '🔸', '🔹', '🔶', '🔷', '🔳', '🔲', '▪️', '▫️', '◾', '◽', '◼️', '◻️', '🟥', '🟧', '🟨', '🟩', '🟦', '🟪', '⬛', '⬜', '🟫', '🔈', '🔇', '🔉', '🔊', '🔔', '🔕', '📣', '📢', '💬', '💭', '🗯️', '♠️', '♣️', '♥️', '♦️', '🃏', '🎴', '🀄', '🕐', '🕑', '🕒', '🕓', '🕔', '🕕', '🕖', '🕗', '🕘', '🕙', '🕚', '🕛', '🕜', '🕝', '🕞', '🕟', '🕠', '🕡', '🕢', '🕣', '🕤', '🕥', '🕦', '🕧']
    },
    {
      id: 'flags',
      name: 'Flags',
      icon: '🏁',
      emojis: ['🏳️', '🏴', '🏴‍☠️', '🏁', '🚩', '🎌', '🏳️‍🌈', '🏳️‍⚧️', '🇺🇳', '🇦🇫', '🇦🇽', '🇦🇱', '🇩🇿', '🇦🇸', '🇦🇩', '🇦🇴', '🇦🇮', '🇦🇶', '🇦🇬', '🇦🇷', '🇦🇲', '🇦🇼', '🇦🇺', '🇦🇹', '🇦🇿', '🇧🇸', '🇧🇭', '🇧🇩', '🇧🇧', '🇧🇾', '🇧🇪', '🇧🇿', '🇧🇯', '🇧🇲', '🇧🇹', '🇧🇴', '🇧🇦', '🇧🇼', '🇧🇷', '🇮🇴', '🇻🇬', '🇧🇳', '🇧🇬', '🇧🇫', '🇧🇮', '🇰🇭', '🇨🇲', '🇨🇦', '🇮🇨', '🇨🇻', '🇧🇶', '🇰🇾', '🇨🇫', '🇹🇩', '🇨🇱', '🇨🇳', '🇨🇽', '🇨🇨', '🇨🇴', '🇰🇲', '🇨🇬', '🇨🇩', '🇨🇰', '🇨🇷', '🇨🇮', '🇭🇷', '🇨🇺', '🇨🇼', '🇨🇾', '🇨🇿', '🇩🇰', '🇩🇯', '🇩🇲', '🇩🇴', '🇪🇨', '🇪🇬', '🇸🇻', '🇬🇶', '🇪🇷', '🇪🇪', '🇸🇿', '🇪🇹', '🇪🇺', '🇫🇰', '🇫🇴', '🇫🇯', '🇫🇮', '🇫🇷', '🇬🇫', '🇵🇫', '🇹🇫', '🇬🇦', '🇬🇲', '🇬🇪', '🇩🇪', '🇬🇭', '🇬🇮', '🇬🇷', '🇬🇱', '🇬🇩', '🇬🇵', '🇬🇺', '🇬🇹', '🇬🇬', '🇬🇳', '🇬🇼', '🇬🇾', '🇭🇹', '🇭🇳', '🇭🇰', '🇭🇺', '🇮🇸', '🇮🇳', '🇮🇩', '🇮🇷', '🇮🇶', '🇮🇪', '🇮🇲', '🇮🇱', '🇮🇹', '🇯🇲', '🇯🇵', '🎌', '🇯🇪', '🇯🇴', '🇰🇿', '🇰🇪', '🇰🇮', '🇽🇰', '🇰🇼', '🇰🇬', '🇱🇦', '🇱🇻', '🇱🇧', '🇱🇸', '🇱🇷', '🇱🇾', '🇱🇮', '🇱🇹', '🇱🇺', '🇲🇴', '🇲🇬', '🇲🇼', '🇲🇾', '🇲🇻', '🇲🇱', '🇲🇹', '🇲🇭', '🇲🇶', '🇲🇷', '🇲🇺', '🇾🇹', '🇲🇽', '🇫🇲', '🇲🇩', '🇲🇨', '🇲🇳', '🇲🇪', '🇲🇸', '🇲🇦', '🇲🇿', '🇲🇲', '🇳🇦', '🇳🇷', '🇳🇵', '🇳🇱', '🇳🇨', '🇳🇿', '🇳🇮', '🇳🇪', '🇳🇬', '🇳🇺', '🇳🇫', '🇰🇵', '🇲🇰', '🇲🇵', '🇳🇴', '🇴🇲', '🇵🇰', '🇵🇼', '🇵🇸', '🇵🇦', '🇵🇬', '🇵🇾', '🇵🇪', '🇵🇭', '🇵🇳', '🇵🇱', '🇵🇹', '🇵🇷', '🇶🇦', '🇷🇪', '🇷🇴', '🇷🇺', '🇷🇼', '🇼🇸', '🇸🇲', '🇸🇹', '🇸🇦', '🇸🇳', '🇷🇸', '🇸🇨', '🇸🇱', '🇸🇬', '🇸🇽', '🇸🇰', '🇸🇮', '🇬🇸', '🇸🇧', '🇸🇴', '🇿🇦', '🇰🇷', '🇸🇸', '🇪🇸', '🇱🇰', '🇧🇱', '🇸🇭', '🇰🇳', '🇱🇨', '🇵🇲', '🇻🇨', '🇸🇩', '🇸🇷', '🇸🇪', '🇨🇭', '🇸🇾', '🇹🇼', '🇹🇯', '🇹🇿', '🇹🇭', '🇹🇱', '🇹🇬', '🇹🇰', '🇹🇴', '🇹🇹', '🇹🇳', '🇹🇷', '🇹🇲', '🇹🇨', '🇹🇻', '🇻🇮', '🇺🇬', '🇺🇦', '🇦🇪', '🇬🇧', '🏴󠁧󠁢󠁥󠁮󠁧󠁿', '🏴󠁧󠁢󠁳󠁣󠁴󠁿', '🏴󠁧󠁢󠁷󠁬󠁳󠁿', '🇺🇸', '🇺🇾', '🇺🇿', '🇻🇺', '🇻🇦', '🇻🇪', '🇻🇳', '🇼🇫', '🇪🇭', '🇾🇪', '🇿🇲', '🇿🇼']
    }
  ];

  const RECENT_EMOJIS_KEY = 'hearth_recent_emojis';
  const MAX_RECENT_EMOJIS = 36;

  let selectedCategory = 0;
  let searchQuery = '';
  let selectedSkinTone = 0;
  let showSkinTonePicker = false;
  let pickerElement: HTMLDivElement;
  let searchInput: HTMLInputElement;
  let recentEmojis: string[] = [];

  // Load recent emojis from localStorage
  function loadRecentEmojis() {
    try {
      const stored = localStorage.getItem(RECENT_EMOJIS_KEY);
      if (stored) {
        recentEmojis = JSON.parse(stored);
        categories[0].emojis = recentEmojis;
      }
    } catch (err) {
      console.error('[EmojiPicker] Failed to load recent emojis:', err);
      recentEmojis = [];
    }
  }

  // Save recent emojis to localStorage
  function saveRecentEmojis() {
    try {
      localStorage.setItem(RECENT_EMOJIS_KEY, JSON.stringify(recentEmojis));
    } catch (err) {
      console.error('[EmojiPicker] Failed to save recent emojis:', err);
    }
  }

  // Add emoji to recent
  function addToRecent(emoji: string) {
    // Remove emoji if it already exists
    recentEmojis = recentEmojis.filter(e => e !== emoji);
    // Add to beginning
    recentEmojis.unshift(emoji);
    // Limit to max
    if (recentEmojis.length > MAX_RECENT_EMOJIS) {
      recentEmojis = recentEmojis.slice(0, MAX_RECENT_EMOJIS);
    }
    categories[0].emojis = recentEmojis;
    saveRecentEmojis();
  }

  // Apply skin tone to emoji
  function applySkintone(emoji: string): string {
    if (selectedSkinTone === 0 || !skinToneEmojis.has(emoji)) {
      return emoji;
    }
    // Remove any existing skin tone modifier and add new one
    const baseEmoji = emoji.replace(/[\u{1F3FB}-\u{1F3FF}]/gu, '');
    return baseEmoji + skinTones[selectedSkinTone].modifier;
  }

  $: filteredEmojis = searchQuery
    ? categories.slice(1).flatMap(c => c.emojis).filter(e =>
        e.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : categories[selectedCategory].emojis;

  $: hasRecentEmojis = categories[0].emojis.length > 0;

  function selectEmoji(emoji: string) {
    const finalEmoji = applySkintone(emoji);
    addToRecent(finalEmoji);
    dispatch('select', finalEmoji);
  }

  function handleClickOutside(event: MouseEvent) {
    if (show && pickerElement && !pickerElement.contains(event.target as Node)) {
      dispatch('close');
    }
  }

  function getEmojiButtons(): HTMLElement[] {
    if (!emojisContainer) return [];
    return Array.from(emojisContainer.querySelectorAll<HTMLElement>('.emoji-btn'));
  }

  function focusEmojiAt(index: number) {
    const buttons = getEmojiButtons();
    if (buttons[index]) {
      buttons[index].focus();
      focusedEmojiIndex = index;
    }
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      if (showSkinTonePicker) {
        showSkinTonePicker = false;
      } else {
        dispatch('close');
      }
      return;
    }
  }

  function handleGridKeydown(event: KeyboardEvent) {
    const buttons = getEmojiButtons();
    if (buttons.length === 0) return;

    // If not focused on grid yet, don't handle
    if (focusedEmojiIndex < 0) return;

    const { handled, newIndex } = handleListKeyboard(event, focusedEmojiIndex, buttons.length, {
      wrap: false,
      gridNavigation: true,
      gridColumns: GRID_COLUMNS,
      onSelect: (idx) => {
        const emoji = filteredEmojis[idx];
        if (emoji) selectEmoji(emoji);
      },
      onEscape: () => dispatch('close')
    });

    if (handled && newIndex !== focusedEmojiIndex) {
      focusEmojiAt(newIndex);
    }
  }

  function handleSearchKeydown(event: KeyboardEvent) {
    // Arrow down from search moves to first emoji
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      focusedEmojiIndex = 0;
      tick().then(() => focusEmojiAt(0));
    } else if (event.key === 'Escape') {
      dispatch('close');
    }
  }

  function handleEmojiFocus(index: number) {
    focusedEmojiIndex = index;
  }

  function selectSkinTone(index: number) {
    selectedSkinTone = index;
    showSkinTonePicker = false;
  }

  onMount(() => {
    loadRecentEmojis();
    document.addEventListener('click', handleClickOutside);
    document.addEventListener('keydown', handleKeydown);

    // Focus search input when opened
    if (searchInput) {
      searchInput.focus();
    }
  });

  onDestroy(() => {
    document.removeEventListener('click', handleClickOutside);
    document.removeEventListener('keydown', handleKeydown);
  });

  // Focus search when picker becomes visible
  $: if (show && searchInput) {
    setTimeout(() => searchInput?.focus(), 0);
  }
</script>

{#if show}
  <div bind:this={pickerElement} class="emoji-picker" role="dialog" aria-label="Emoji picker" aria-modal="true">
    <!-- Header with search and skin tone -->
    <div class="header">
      <div class="search-container">
        <svg class="search-icon" viewBox="0 0 24 24" width="16" height="16" aria-hidden="true">
          <path fill="currentColor" d="M21.707 20.293l-4.054-4.054A8.46 8.46 0 0 0 19.5 11c0-4.687-3.813-8.5-8.5-8.5S2.5 6.313 2.5 11s3.813 8.5 8.5 8.5a8.46 8.46 0 0 0 5.239-1.847l4.054 4.054a1 1 0 0 0 1.414-1.414zM11 17.5c-3.584 0-6.5-2.916-6.5-6.5S7.416 4.5 11 4.5s6.5 2.916 6.5 6.5-2.916 6.5-6.5 6.5z"/>
        </svg>
        <input
          bind:this={searchInput}
          type="text"
          placeholder="Search emoji"
          bind:value={searchQuery}
          class="search-input"
          aria-label="Search emoji"
          on:keydown={handleSearchKeydown}
        />
      </div>

      <!-- Skin tone selector -->
      <div class="skin-tone-container">
        <button
          class="skin-tone-button"
          on:click|stopPropagation={() => showSkinTonePicker = !showSkinTonePicker}
          title="Select skin tone"
          aria-label="Select skin tone: {skinTones[selectedSkinTone].name}"
          aria-expanded={showSkinTonePicker}
          aria-haspopup="listbox"
          type="button"
        >
          <span class="skin-tone-preview" style="background-color: {skinTones[selectedSkinTone].color}" aria-hidden="true"></span>
        </button>

        {#if showSkinTonePicker}
          <div class="skin-tone-picker" role="listbox" aria-label="Skin tone options">
            {#each skinTones as tone, i}
              <button
                class="skin-tone-option"
                class:selected={selectedSkinTone === i}
                on:click|stopPropagation={() => selectSkinTone(i)}
                title={tone.name}
                role="option"
                aria-selected={selectedSkinTone === i}
                aria-label="{tone.name} skin tone"
                type="button"
              >
                <span class="skin-tone-swatch" style="background-color: {tone.color}" aria-hidden="true"></span>
              </button>
            {/each}
          </div>
        {/if}
      </div>
    </div>

    <!-- Category tabs -->
    <div class="categories" role="tablist" aria-label="Emoji categories">
      {#each categories as category, i}
        {#if i > 0 || hasRecentEmojis}
          <button
            class="category-btn"
            class:active={selectedCategory === i && !searchQuery}
            on:click={() => { selectedCategory = i; searchQuery = ''; }}
            title={category.name}
            role="tab"
            aria-selected={selectedCategory === i && !searchQuery}
            aria-label={category.name}
            type="button"
          >
            <span aria-hidden="true">{category.icon}</span>
          </button>
        {/if}
      {/each}
    </div>

    <!-- Category label -->
    {#if !searchQuery}
      <div class="category-label">
        {categories[selectedCategory].name}
      </div>
    {:else}
      <div class="category-label">
        Search Results
      </div>
    {/if}

    <!-- Emoji grid -->
    <div 
      bind:this={emojisContainer}
      class="emojis" 
      role="grid" tabindex="0" 
      aria-label="{searchQuery ? 'Search Results' : categories[selectedCategory].name} emoji"
      on:keydown={handleGridKeydown}
    >
      {#each filteredEmojis as emoji, i}
        <button
          class="emoji-btn"
          on:click={() => selectEmoji(emoji)}
          on:focus={() => handleEmojiFocus(i)}
          title={emoji}
          aria-label="Select {emoji} emoji"
          type="button"
          tabindex={focusedEmojiIndex === i ? 0 : -1}
        >
          {applySkintone(emoji)}
        </button>
      {/each}
      {#if filteredEmojis.length === 0}
        <div class="no-results">
          {#if searchQuery}
            No emoji found for "{searchQuery}"
          {:else}
            No recent emoji
          {/if}
        </div>
      {/if}
    </div>

    <!-- Footer with emoji preview -->
    <div class="footer">
      <div class="preview-emoji">
        {#if filteredEmojis.length > 0}
          {applySkintone(filteredEmojis[0])}
        {:else}
          😀
        {/if}
      </div>
      <div class="preview-info">
        <span class="preview-name">
          {#if filteredEmojis.length > 0}
            Hover to preview
          {:else}
            No emoji selected
          {/if}
        </span>
      </div>
    </div>
  </div>
{/if}

<style>
  .emoji-picker {
    position: absolute;
    bottom: 100%;
    right: 0;
    width: 418px;
    height: 445px;
    background-color: var(--bg-floating, #2f3136);
    border-radius: 8px;
    box-shadow: 0 8px 16px rgba(0, 0, 0, 0.24);
    display: flex;
    flex-direction: column;
    z-index: 100;
    margin-bottom: 8px;
    animation: pickerSlideIn 0.15s ease-out;
  }

  @keyframes pickerSlideIn {
    from {
      opacity: 0;
      transform: translateY(8px) scale(0.98);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }

  .header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px;
    border-bottom: 1px solid var(--bg-modifier-accent, #3f4147);
  }

  .search-container {
    flex: 1;
    position: relative;
    display: flex;
    align-items: center;
  }

  .search-icon {
    position: absolute;
    left: 10px;
    color: var(--text-muted, #b5bac1);
    pointer-events: none;
  }

  .search-input {
    width: 100%;
    padding: 8px 12px 8px 34px;
    background-color: var(--bg-tertiary, #1e1f22);
    border: none;
    border-radius: 4px;
    color: var(--text-normal, #f2f3f5);
    font-size: 14px;
  }

  .search-input::placeholder {
    color: var(--text-muted, #b5bac1);
  }

  .search-input:focus {
    outline: none;
  }

  .search-input:focus-visible {
    outline: 2px solid var(--blurple, #5865f2);
    outline-offset: -2px;
  }

  .skin-tone-container {
    position: relative;
  }

  .skin-tone-button {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    background: transparent;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    transition: background-color 0.15s;
  }

  .skin-tone-button:hover {
    background-color: var(--bg-modifier-hover, #35373c);
  }

  .skin-tone-button:focus-visible {
    outline: 2px solid var(--blurple, #5865f2);
    outline-offset: 2px;
  }

  .skin-tone-preview {
    width: 20px;
    height: 20px;
    border-radius: 50%;
    border: 2px solid var(--bg-modifier-accent, #3f4147);
  }

  .skin-tone-picker {
    position: absolute;
    top: 100%;
    right: 0;
    display: flex;
    gap: 4px;
    padding: 8px;
    background-color: var(--bg-floating, #18191c);
    border-radius: 4px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.32);
    z-index: 10;
    margin-top: 4px;
  }

  .skin-tone-option {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    padding: 0;
    background: transparent;
    border: 2px solid transparent;
    border-radius: 4px;
    cursor: pointer;
    transition: border-color 0.15s;
  }

  .skin-tone-option:hover {
    border-color: var(--bg-modifier-accent, #3f4147);
  }

  .skin-tone-option.selected {
    border-color: var(--blurple, #5865f2);
  }

  .skin-tone-swatch {
    width: 20px;
    height: 20px;
    border-radius: 50%;
  }

  .categories {
    display: flex;
    padding: 0 8px;
    gap: 2px;
    border-bottom: 1px solid var(--bg-modifier-accent, #3f4147);
    background-color: var(--bg-secondary, #2b2d31);
  }

  .category-btn {
    flex: 1;
    padding: 8px 4px;
    background: transparent;
    border: none;
    border-bottom: 2px solid transparent;
    font-size: 18px;
    cursor: pointer;
    opacity: 0.5;
    transition: opacity 0.15s, border-color 0.15s;
  }

  .category-btn:hover {
    opacity: 0.8;
  }

  .category-btn:focus-visible {
    outline: 2px solid var(--blurple, #5865f2);
    outline-offset: -2px;
    opacity: 1;
  }

  .category-btn.active {
    opacity: 1;
    border-bottom-color: var(--blurple, #5865f2);
  }

  .category-label {
    padding: 8px 12px 4px;
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
    color: var(--text-muted, #b5bac1);
  }

  .emojis {
    flex: 1;
    overflow-y: auto;
    padding: 0 8px 8px;
    display: grid;
    grid-template-columns: repeat(9, 1fr);
    gap: 2px;
    align-content: start;
  }

  .emoji-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 6px;
    background: transparent;
    border: none;
    border-radius: 4px;
    font-size: 22px;
    cursor: pointer;
    transition: background-color 0.1s, transform 0.1s;
  }

  .emoji-btn:hover {
    background-color: var(--bg-modifier-hover, #35373c);
    transform: scale(1.15);
  }

  .emoji-btn:focus-visible {
    outline: 2px solid var(--blurple, #5865f2);
    outline-offset: -2px;
    background-color: var(--bg-modifier-hover, #35373c);
  }

  .no-results {
    grid-column: 1 / -1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 32px 16px;
    text-align: center;
    color: var(--text-muted, #b5bac1);
    font-size: 14px;
  }

  .footer {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 12px;
    background-color: var(--bg-secondary, #2b2d31);
    border-top: 1px solid var(--bg-modifier-accent, #3f4147);
    border-radius: 0 0 8px 8px;
  }

  .preview-emoji {
    font-size: 28px;
    line-height: 1;
  }

  .preview-info {
    display: flex;
    flex-direction: column;
  }

  .preview-name {
    font-size: 13px;
    color: var(--text-muted, #b5bac1);
  }
</style>
