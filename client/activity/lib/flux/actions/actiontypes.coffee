keyMirror = require 'keymirror'

module.exports = keyMirror({
  'LOAD_MESSAGES_BEGIN'
  'LOAD_MESSAGES_FAIL'
  'LOAD_MESSAGE_SUCCESS'

  'CREATE_MESSAGE_BEGIN'
  'CREATE_MESSAGE_SUCCESS'
  'CREATE_MESSAGE_FAIL'

  'REMOVE_MESSAGE_BEGIN'
  'REMOVE_MESSAGE_SUCCESS'
  'REMOVE_MESSAGE_FAIL'

  'LIKE_MESSAGE_BEGIN'
  'LIKE_MESSAGE_SUCCESS'
  'LIKE_MESSAGE_FAIL'

  'UNLIKE_MESSAGE_BEGIN'
  'UNLIKE_MESSAGE_SUCCESS'
  'UNLIKE_MESSAGE_FAIL'

  'CHANGE_SELECTED_CHANNEL'

})

