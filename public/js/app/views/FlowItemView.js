/*global define */

define([
    'utils',
    'marionette',
    'templates',
    'models/Dialog'
], function (Utils, Marionette, templates, DialogModel) {
    'use strict';

    return Marionette.ItemView.extend({
        template: templates.flowItem,
        tagName: 'div',
        className: 'row',

        events: {
            'click a': 'activateMenu',
            'click button.play': 'playButtonClick',
            'click button.stop': 'stopButtonClick'
        },
        modelEvents: {
            "change:active": function() {
                this.render();
            }
        },
        
        lastResults: null,  // set if we have a results status

        taskDialogModel: null,  // set when we have had an open dialog

        onShow: function() {
            var self = this
            var g = new dagreD3.Digraph();

            var nodes = this.model.get("Nodes");
            var edges = this.model.get("Edges");
            
            var len = nodes.length;
            for (var i = 0; i < len; i++) {
                var n = nodes[i];
                g.addNode(n.Id,    { label: '<div><div class="task-status"></div></div><p>' + n.Name + '</p><p class="small">(' + n.Type + ")</p>"});
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
                }).on('click', function(id){
                    console.log("click - " + id);
                    self.showTaskModal(id);
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

            self.grabLatest(this.model.get("Id"));

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


        stopButtonClick: function(e) {
            var self = this;
            
            var id = this.model.get("Id")

            console.log("stop clicked on "+ id)

            var pl = {
              Id: id
            };

            Utils.sendObj(pl, app.API_ROOT + '/api/stop', function (resp){
                self.playLoop(id);
            }, function (resp) {
                console.log(resp);
            });

            
            // after 5 seconds stop the loop if it didnt already
            setTimeout(function() {
                self.stop = true;
            }, 5000);
        },

        stop: false,

        playLoop: function(id) {
            var self = this;
            self.stop = false;
            // 5 mins 
            setTimeout(function() {
                stopLoop();
            }, 300000);

            var loop = setInterval(function() {
                self.grabLatest(id);
                if (self.stop) {
                    stopLoop();
                }
            }, 500);

            var pulser = setInterval(function(){
                // pulse in-progress
                
                $(".in-progress .task-status").addClass("task-status-fade")

                setTimeout(function() {
                    $(".in-progress .task-status").removeClass("task-status-fade")
                }, 300)
                
            }, 1000)

            function stopLoop() {
                clearInterval(loop)
                clearInterval(pulser)
            }
        },

        grabLatest: function(id) {
            var self = this;
            Utils.getObj(app.API_ROOT + '/api/status/current', function (resp) {

                var res = resp[id];
                if (res == null) {
                    return 
                }

                if (res.Error != "" ) {
                    self.stop = true;
                    app.commands.execute("app:dialog:simple", {
                        icon: 'info-sign',    // Optional. default is (glyphicon-)bell
                        title: 'Problem!', // Optional
                        message: res.Error
                    });

                    return
                }

                res = res.Results;
                // store them
                self.lastResults = res;

                console.log(res);

                for (var key in res) {
                    console.log(key);

                    // update any active dialog model
                    if (self.taskDialogModel) {
                        if(self.taskDialogModel.key == key) {
                            var obj = self.enhanceStatus(key, res[key])
                            self.taskDialogModel.model.set(obj);
                        }
                    }

                    var node = $('#node-' + key + ' div').first();
                    if (res[key].PercentComplete == 100) {
                        node.addClass('success');
                        node.removeClass('in-progress');
                        node.find(".task-status").removeClass("task-status-fade")
                        if (res[key].Failed > 0) {
                            node.addClass('failure');
                        }
                    } else if (res[key].PercentComplete > 0) {
                        node.removeClass('success');
                        node.removeClass('failure');
                        node.addClass('in-progress');
                    } else {
                        node.removeClass('success');
                        node.removeClass('failure');
                        node.removeClass('in-progress');
                        node.find(".task-status").removeClass("task-status-fade")
                    }
                }

                if (resp[id].Completed) {
                    self.stop = true;
                }

            }, function (resp) {
                console.log(resp);
            });    
        },

        enhanceStatus: function(id, inobj) {
            var obj = inobj 
            obj.title = id;
            obj.message = "Complete"
            if (obj.Failed > 0 ) {
                obj.message = "Problems"
            } else if (obj.PercentComplete == 0) {
                obj.message = "Not Started"
            } else if (obj.PercentComplete < 100) {
                obj.message = "In Progress"
            }

            obj.RenderedOutput = "";
            for (var n in obj.CommandOutput) {
                var line = obj.CommandOutput[n];
                obj.RenderedOutput = obj.RenderedOutput + " " + _.escape($.trim(line)) + "\n";
            }

            return obj;
        },

        showTaskModal: function(id) {
            var obj = {
                title: id,
                message: "No run data recorded for this task",
                CommandOutput: null,
                RenderedOutput: "",
                CommandStream: Object,
                Complete: 10,
                Failed: 0,
                PercentComplete: 100,
            }
               
            if (this.lastResults) {
                obj = this.enhanceStatus(id, this.lastResults[id])
            }

            this.taskDialogModel = {key: id, model: new DialogModel(obj)}
            
            app.commands.execute("app:dialog:taskstatus", this.taskDialogModel.model);
        },
    });
});
