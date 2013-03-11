class MainView extends KDView

  viewAppended:->

    @mc = @getSingleton 'mainController'
    @addHeader()
    @createMainPanels()
    @createMainTabView()
    @createSideBar()
    @listenWindowResize()

  addBook:-> @addSubView new BookView

  setViewState:(state)->

    switch state
      when 'hideTabs'
        @contentPanel.setClass 'no-shadow'
        @mainTabView.hideHandleContainer()
        @sidebar.hideFinderPanel()
      when 'application'
        @contentPanel.unsetClass 'no-shadow'
        @mainTabView.showHandleContainer()
        @sidebar.showFinderPanel()
      else
        @contentPanel.unsetClass 'no-shadow'
        @mainTabView.showHandleContainer()
        @sidebar.hideFinderPanel()

  removeLoader:->
    $loadingScreen = $(".main-loading").eq(0)
    {winWidth,winHeight} = @getSingleton "windowController"
    $loadingScreen.css
      marginTop : -winHeight
      opacity   : 0
    @utils.wait 601, =>
      $loadingScreen.remove()
      $('body').removeClass 'loading'

  createMainPanels:->

    @addSubView @panelWrapper = new KDView
      tagName  : "section"

    @panelWrapper.addSubView @sidebarPanel = new KDView
      domId    : "sidebar-panel"

    @panelWrapper.addSubView @contentPanel = new KDView
      domId    : "content-panel"
      cssClass : "transition"
      bind     : "webkitTransitionEnd" #TODO: Cross browser support

    @contentPanel.on "ViewResized", (rest...)=> @emit "ContentPanelResized", rest...

    @registerSingleton "contentPanel", @contentPanel, yes
    @registerSingleton "sidebarPanel", @sidebarPanel, yes

    @contentPanel.on "webkitTransitionEnd", (e) =>
      @emit "mainViewTransitionEnd", e

  addHeader:()->

    @addSubView @header = new KDView
      tagName : "header"

    @header.addSubView @logo = new KDCustomHTMLView
      tagName   : "a"
      domId     : "koding-logo"
      # cssClass  : "hidden"
      attributes:
        href    : "#"
      click     : (event)=>
        return if @userEnteredFromGroup()

        event.stopPropagation()
        event.preventDefault()
        KD.getSingleton('router').handleRoute null

  createMainTabView:->

    @mainTabHandleHolder = new MainTabHandleHolder
      domId    : "main-tab-handle-holder"
      cssClass : "kdtabhandlecontainer"
      delegate : @

    getFrontAppManifest = ->
      appManager = KD.getSingleton "appManager"
      appController = KD.getSingleton "kodingAppsController"
      frontApp = appManager.getFrontApp()
      frontAppName = name for name, instances of appManager.appControllers when frontApp in instances
      appController.constructor.manifests?[frontAppName]

    @mainSettingsMenuButton = new KDButtonView
      domId    : "main-settings-menu"
      cssClass : "kdsettingsmenucontainer transparent"
      iconOnly : yes
      iconClass: "dot"
      callback : ->
        appManifest = getFrontAppManifest()
        if appManifest?.menu
          appManifest.menu.forEach (item, index)->
            item.callback = (contextmenu)->
              mainView = KD.getSingleton "mainView"
              view = mainView.mainTabView.activePane?.mainView
              item.eventName or= item.title
              view?.emit "menu.#{item.eventName}", item.eventName, item, contextmenu

          offset = @$().offset()
          contextMenu = new JContextMenu
              event       : event
              delegate    : @
              x           : offset.left - 150
              y           : offset.top + 20
              arrow       :
                placement : "top"
                margin    : -5
            , appManifest.menu
    @mainSettingsMenuButton.hide()

    @mainTabView = new MainTabView
      domId              : "main-tab-view"
      listenToFinder     : yes
      delegate           : @
      slidingPanes       : no
      tabHandleContainer : @mainTabHandleHolder
    ,null

    @mainTabView.on "PaneDidShow", => KD.utils.wait 10, =>
      appManifest = getFrontAppManifest()
      @mainSettingsMenuButton[if appManifest?.menu then "show" else "hide"]()

    mainController = @getSingleton('mainController')
    mainController.popupController = new VideoPopupController

    mainController.monitorController = new MonitorController

    @videoButton = new KDButtonView
      cssClass : "video-popup-button"
      icon : yes
      title : "Video"
      callback :=>
        unless @popupList.$().hasClass "hidden"
          @videoButton.unsetClass "active"
          @popupList.hide()
        else
          @videoButton.setClass "active"
          @popupList.show()

    @videoButton.hide()

    @popupList = new VideoPopupList
      cssClass      : "hidden"
      type          : "videos"
      itemClass     : VideoPopupListItem
      delegate      : @
    , {}

    @mainTabView.on "AllPanesClosed", ->
      @getSingleton('router').handleRoute "/Activity"

    @contentPanel.addSubView @mainTabView
    @contentPanel.addSubView @mainTabHandleHolder
    @contentPanel.addSubView @mainSettingsMenuButton
    @contentPanel.addSubView @videoButton
    @contentPanel.addSubView @popupList

    getSticky = =>
      @getSingleton('windowController')?.stickyNotification
    getStatus = =>
      KD.remote.api.JSystemStatus.getCurrentSystemStatus (err,systemStatus)=>
        if err
          if err.message is 'none_scheduled'
            getSticky()?.emit 'restartCanceled'
          else
            log 'current system status:',err
        else
          systemStatus.on 'restartCanceled', =>
            getSticky()?.emit 'restartCanceled'
          new GlobalNotification
            targetDate  : systemStatus.scheduledAt
            title       : systemStatus.title
            content     : systemStatus.content
            type        : systemStatus.type

    # sticky = @getSingleton('windowController')?.stickyNotification
    @utils.defer => getStatus()

    KD.remote.api.JSystemStatus.on 'restartScheduled', (systemStatus)=>
      sticky = @getSingleton('windowController')?.stickyNotification

      if systemStatus.status isnt 'active'
        getSticky()?.emit 'restartCanceled'
      else
        systemStatus.on 'restartCanceled', =>
          getSticky()?.emit 'restartCanceled'
        new GlobalNotification
          targetDate : systemStatus.scheduledAt
          title      : systemStatus.title
          content    : systemStatus.content
          type       : systemStatus.type

  createSideBar:->

    @sidebar = new Sidebar domId : "sidebar", delegate : @
    @emit "SidebarCreated", @sidebar
    @sidebarPanel.addSubView @sidebar

  changeHomeLayout:(isLoggedIn)->

  userEnteredFromGroup:-> KD.config.groupEntryPoint?
  userEnteredFromProfile:-> KD.config.profileEntryPoint?

  switchProfileState:(isLoggedIn)->
    $('body').addClass "login"
    @addProfileViews()
    # @getSingleton('router').handleRoute "/Activity"

  switchGroupState:(isLoggedIn)->

    $('.group-loader').removeClass 'pulsing'
    $('body').addClass "login"

    {groupEntryPoint} = KD.config

    loginLink = new GroupsLandingPageButton {groupEntryPoint}, {}

    loginLink.on 'LoginLinkRedirect', ({section})=>

      route =  "/#{groupEntryPoint}/#{section}"

      switch section
        when 'Join', 'Login'
          @mc.loginScreen.animateToForm 'login'
          @mc.loginScreen.headBannerShowGoBackGroup 'Pet Shop Boys'
          $('#group-landing').css 'height', 0
          # $('#group-landing').css 'opacity', 0

        when 'Activity'
          @mc.loginScreen.hide()
          KD.getSingleton('router').handleRoute route
          $('#group-landing').css 'height', 0

    if isLoggedIn and groupEntryPoint?
      KD.whoami().fetchGroupRoles groupEntryPoint, (err, roles)->
        if err then console.warn err
        else if roles.length
          loginLink.setState { isMember: yes, roles }
        else
          {JMembershipPolicy} = KD.remote.api
          JMembershipPolicy.byGroupSlug groupEntryPoint,
            (err, policy)->
              if err then console.warn err
              else if policy?
                loginLink.setState {
                  isMember        : no
                  approvalEnabled : policy.approvalEnabled
                }
              else
                loginLink.setState {
                  isMember        : no
                  isPublic        : yes
                }
    else
      @utils.defer -> loginLink.setState { isLoggedIn: no }

    loginLink.appendToSelector '.group-login-buttons'

  closeGroupView:->
    @mainTabView.showHandleContainer()
    $('.group-landing').css 'height', 0

  closeProfileView:->
    @mainTabView.showHandleContainer()
    @profileLandingView._windowDidResize = noop
    $('.profile-landing').css 'height', 0

  addProfileViews:->


    @profileLandingView = new KDView
      lazyDomId : 'profile-landing'

    @profileContentView = new KDView
      lazyDomId : 'profile-content'

    @profileContentWrapperView = new KDView
      lazyDomId : 'profile-content-wrapper'
      cssClass : 'slideable'

    @profilePersonalWrapperView = new KDView
      lazyDomId : 'profile-personal-wrapper'
      cssClass : 'slideable'

    @profileLogoView = new KDView
      lazyDomId: 'profile-koding-logo'
      click :=>
        @profilePersonalWrapperView.setClass 'slide-down'
        @profileContentWrapperView.setClass 'slide-down'
        @profileLogoView.setClass 'top'

        @profileLandingView.setClass 'profile-fading'
        @utils.wait 1100, => @profileLandingView.setClass 'profile-hidden'

    @profileLogoView.$().css
      top: @profileLandingView.getHeight()-42

    @utils.wait => @profileLogoView.setClass 'animate'

    KD.remote.cacheable @profileLandingView.$().attr('data-profile'), (err, user, name)=>
      # account = user
      if KD.whoami().getId() is user.getId()
        @profileContentView.addSubView createBlogPostButton = new KDButtonView
          title               : 'Post a new blog entry'
          cssClass            : 'new-blogpost clean-gray'
          callback            : =>
            modal             = new KDModalView
              cssClass        : 'new-blogpost-modal'
              title           : 'Post new blog entry'
              overlay         : yes
              height          : "auto"
              buttons         :
                cancel        :
                  style       : 'modal-cancel'
                  callback    : -> modal.destroy()
                post          :
                  style       : 'modal-clean-gray'
                  callback    : =>
                    KD.remote.api.JBlogPost.create
                      title   : @titleInput.getValue()
                      content : @markdownInput.getValue()
                    , =>
                      modal.buttons.cancel.hideLoader()
                      modal.destroy()
            modal.addSubView formline = new KDView
              cssClass : 'profile-modal formline'
            formline.addSubView @titleInput = new KDInputView
              cssClass : 'title-input'
            formline.addSubView @markdownInput = new KDInputViewWithPreview
              cssClass  : 'markdown-input'
              type : 'textarea'
              preview         :
                showInitially : no


    # KD.remote.cacheable @profileLandingView.$().attr('data-profile'), (err, user, name)=>
    #   if user.skillTags
    #     @profileTagGroupView = new SkillTagGroup
    #       lazyDomId :  'skill-tags'
    #     , user
    #     @profileTagGroupView.on 'TagWasClicked', =>
    #       @closeProfileView()
    #     @profileTagGroupView.viewAppended()

    #   # selector =
    #   #   originId  : user.getId()
    #   #   type      : 'CStatusActivity'
    #   # log selector

    #   # @getSingleton("appManager").tell 'Activity', 'fetchTeasers', selector, {}
    #   # , (data)->
    #   #   log 'teasers?'
    #   #   log data

    #   # @getSingleton("appManager").tell 'Activity', 'fetchCachedActivity', {}
    #   # , (err, data)->
    #   #   log 'cache?'
    #   #   log data

    #   # statusUpdatesWrapper.addSubView  statusUpdatesList = new KDListView
    #   #   itemClass : StatusActivityItemView

    #   @profileContentView.addSubView statusUpdatesWrapper = new KDView
    #     cssClass : 'status-updates profile-wrapper'

    #   @profileContentView.addSubView otherWrapper = new KDView
    #     cssClass : 'other profile-wrapper'

    #   @profileContentView.addSubView footerWrapper = new KDView
    #     cssClass : 'footer profile-wrapper'

    #   statusUpdatesWrapper.addSubView statusUpdatesListHeader = new KDView
    #     cssClass : 'profile-list'
    #     partial : 'I am a Status Update list'


    #   otherWrapper.addSubView otherList = new KDView
    #     cssClass : 'profile-list'
    #     partial : 'I am a list of other things'

    #   footerWrapper.addSubView footerList = new KDView
    #     cssClass : 'profile-list'
    #     partial : 'I am a list of footer'


    #   @profileContentView.setClass 'ready'
    #   @profileContentView.render()

    @profileLandingView.listenWindowResize()

    @profileLandingView._windowDidResize = =>
      @profileLandingView?.setHeight window.outerHeight

  decorateLoginState:(isLoggedIn = no)->

    groupLandingView = new KDView
      lazyDomId : 'group-landing'

    groupLandingView.listenWindowResize()
    groupLandingView._windowDidResize = =>
      groupLandingView.setHeight window.innerHeight - 50

    if isLoggedIn
      if @userEnteredFromGroup()
        @switchGroupState yes
      else if @userEnteredFromProfile()
        @switchProfileState yes
      else
        $('body').addClass "loggedIn"
        @mainTabView.showHandleContainer()

      logoutLinkView = new LandingPageNavLink
        title : 'Logout'

      @mainTabView.showHandleContainer()
      @contentPanel.setClass "social"  if "Develop" isnt @getSingleton("router")?.getCurrentPath()
      # @buttonHolder.hide()

    else
      if @userEnteredFromGroup()
        @switchGroupState no
      else if @userEnteredFromProfile()
        @switchProfileState no
      else
        $('body').removeClass "loggedIn"

      loginLinkView = new LandingPageNavLink
        title : 'Login'

      @contentPanel.unsetClass "social"
      @mainTabView.hideHandleContainer()
      # @buttonHolder.show()

    @changeHomeLayout isLoggedIn
    @utils.wait 300, => @notifyResizeListeners()

  _windowDidResize:->

    {winHeight} = @getSingleton "windowController"
    @panelWrapper.setHeight winHeight - 51
