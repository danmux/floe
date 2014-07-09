/*global define */

define([
    'backbone',
    'utils',
    'app',
    'views/HomeView',
    'views/PageView'
], function (Backbone, Utils, app, HomeView, PageView) {
    'use strict';

    return {
        showPage: function (pageName) {
            if(pageName == null) pageName = 'dash';

            console.log('Router => Showing page: ' + pageName);
            var pageModel = app.pages.findWhere({name: pageName});

            app.vent.trigger('menu:activate', pageModel);
            if(pageName == 'dash') {


                Utils.getObj(app.API_ROOT + '/api/flow', function (response) {

                    var collection = new Backbone.Collection(response.Flows);

                    app.main.show(new HomeView({
                        model: pageModel, 
                        collection: collection
                    }));

                }, function (thing){

                });
                
            } else {
                app.main.show(new PageView({model: pageModel}));
            }

            if(pageName == 'about') {
                console.log('Example of on demand module loading..');
                require(['modules/Example'], function(Example) {
                    Example.start();
                });
            }
        },
        hello: function() {
            console.log('In route /hi');
        }
    };
});
