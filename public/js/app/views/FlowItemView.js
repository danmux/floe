/*global define */

define([
    'utils',
    'marionette',
    'templates',
    'models/Page',
], function (Utils, Marionette, templates) {
    'use strict';

    return Marionette.ItemView.extend({
        template: templates.flowItem,
        tagName: 'div',
        className: 'row',

        events: {
            'click a': 'activateMenu',
            'click button.play': 'playButtonClick'
        },
        modelEvents: {
            "change:active": function() {
                this.render();
            }
        },
        onShow: function() {
            var g = new dagreD3.Digraph();

            var nodes = this.model.get("Nodes");
            var edges = this.model.get("Edges");
            
            var len = nodes.length;
            for (var i = 0; i < len; i++) {
                var n = nodes[i];
                g.addNode(n.Name,    { label: "<p>"+n.Name + '</p><p class="small">(' + n.Type + ")</p>"});
            }

            len = edges.length;
            for (var i = 0; i < len; i++) {
                var n = edges[i];
                g.addEdge(null, n.From, n.To, { label: n.Name});
            }


            var renderer = new dagreD3.Renderer();

            var oldDrawNodes = renderer.drawNodes();
          
            renderer.drawNodes(function(graph, root) {
                var svgNodes = oldDrawNodes(graph, root);
                // todo - can we get the id
                svgNodes.attr("id", function(u) { 
                    var p = u.toLowerCase().split(" ")
                    return "node-" + p.join('-'); 
                });
                return svgNodes;
             });

          
              // d3.select("svg")
              //   .attr("width", layout.graph().width + 40)
              //   .attr("height", layout.graph().height + 40);

            var layout = dagreD3.layout()
                                .nodeSep(20)
                                .rankDir("LR");
            renderer.layout(layout).run(g, d3.select("#" + this.model.get("Id") + " svg g"));
        },

        playButtonClick: function(e) {
            var self = this;
            // e.preventDefault();
            // e.stopPropagation();
            var id = this.model.get("Id")

            console.log("play clicked on "+ id)

            var pl = {
              Id: id,
              Command: "exec",
              Delay: 2
            };

            Utils.sendObj(pl, app.API_ROOT + '/api/exec', function (resp){
                self.playLoop(id);
            }, function (resp) {
                console.log(resp);
            });
        },

        playLoop: function(id) {
            var stop = false;
            // 10 seconds max job for now
            setTimeout(function() {
                stopLoop();
            }, 20000);

            var loop = setInterval(function() {
                Utils.getObj(app.API_ROOT + '/api/status/current', function (resp){

                    var res = resp[id].Results

                    console.log(res);

                    for (var key in res) {
                        console.log(key);
                        $('#node-' + key + ' div').addClass('success')
                    }

                    if (resp[id].Completed) {
                        stopLoop();
                    }
                }, function (resp) {
                    console.log(resp);
                });    
            }, 500);

            function stopLoop() {
                clearInterval(loop)
            }
        }
    });
});
