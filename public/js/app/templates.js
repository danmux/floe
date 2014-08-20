/*global define */

define(function (require) {
	'use strict';

	return {
        pages : {
          dash: require('tpl!templates/pages/dash.html'),
          settings: require('tpl!templates/pages/settings.html'),
          agents: require('tpl!templates/pages/agents.html')
        },
        flowItem: require('tpl!templates/flowItem.html'),
        page: require('tpl!templates/page.html'),
        menuItem: require('tpl!templates/menuItem.html'),
		    footer: require('tpl!templates/footer.html')
	};
});

