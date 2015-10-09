kd                      = require 'kd'
KDView                  = kd.View
KDInputView             = kd.InputView
KDButtonView            = kd.ButtonView
KDLoaderView            = kd.LoaderView
KDCustomHTMLView        = kd.CustomHTMLView


module.exports = class InvitationInputView extends KDView

  constructor: (options = {}, data) ->

    options.cssClass     = 'invite-inputs'
    options.cancellable ?= yes

    super options, data

    @createElements()


  createElements: ->

    @addSubView @email = new KDInputView
      cssClass     : 'user-email'
      placeholder  : 'mail@example.com'
      validate     :
        rules      :
          required : yes
          email    : yes

    @addSubView @firstName = new KDInputView
      cssClass    : 'firstname'
      placeholder : 'Optional'

    @addSubView @lastName = new KDInputView
      cssClass    : 'lastname'
      placeholder : 'Optional'

    if cancellable
      @addSubView @cancel = new KDCustomHTMLView
        tagName  : 'span'
        cssClass : 'cancel icon'
        click    : => @destroy()

    @inputs = [ @email, @firstName, @lastName ]


  serialize: ->

    return {
      email     : @email.getValue()
      firstName : @firstName.getValue()
      lastName  : @lastName.getValue()
    }

