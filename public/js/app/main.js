require.config({
	paths: {
		underscore: '../lib/underscore/underscore',
		backbone: '../lib/backbone/backbone',
		marionette: '../lib/backbone.marionette/lib/backbone.marionette',
		jquery: '../lib/jquery/jquery',
		localStorage: '../lib/backbone.localStorage/backbone.localStorage',
		tpl: '../lib/tpl',
        bootstrap: '../lib/bootstrap.min'
	},

	shim: {
		underscore: {
			exports: '_'
		},

		backbone: {
			exports: 'Backbone',
			deps: ['jquery', 'underscore']
		},

		marionette: {
			exports: 'Backbone.Marionette',
			deps: ['backbone']
		},

        bootstrap: {
            deps: ['jquery']
        }

	},
    waitSeconds: 60
});

require([
	'app',
    'modules/Pages',
    'jquery',
	'bootstrap'
], function (app, PagesModule) {
	'use strict';

    app.addInitializer(function() {
        PagesModule.start();
    });

	app.start();
});
