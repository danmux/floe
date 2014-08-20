/*global define */

define([
	'marionette'
], function (Marionette) {
	'use strict';

	return Marionette.ItemView.extend({
		events: {
            'click .dismiss': 'dismiss',
        },
        modelEvents: {
            'change': 'render'
        },

        onShow: function() {
            $('pre code').each(function(i, block) {
                hljs.highlightBlock(block);
            });
        },  

        dismiss: function(e) {
            e.preventDefault();
            this.trigger('dialog:close');
        }
	});
});
