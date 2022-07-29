import Ember from 'ember';

export default Ember.Controller.extend({
  applicationController: Ember.inject.controller('application'),
  netstats: Ember.computed.reads('applicationController'),
  stats: Ember.computed.reads('applicationController.model.stats'),
  config: Ember.computed.reads('applicationController.config'),
  hashrate: Ember.computed.reads('applicationController.hashrate'),

  chartOptions: Ember.computed("model.hashrate", {
        get() {
            var e = this,
                t = e.getWithDefault("model.minerCharts"),
                a = {
                    chart: {
                        backgroundColor: "rgba(255, 255, 255, 0.1)",
                        type: "spline",
                        marginRight: 10,
                        height: 290,
                        events: {
                            load: function() {
                                var series = this.series[0];
                                setInterval(function() {
                                    var x = (new Date()).getTime(),
                                        y = e.getWithDefault("model.currentHashrate") / 1000000;
                                    series.addPoint([x, y], true, true);
                                }, 1090000000);
                            }
                        }
                    },
                    title: {
                        text: ""
                    },
                  //////
                  rangeSelector : {
                enabled: true,
               selected: 4,
      allButtonsEnabled: true,
      inputDateFormat: '%Y/%m/%d %H:%M',
      inputEditDateFormat: '%Y/%m/%d %H:%M',
      inputEnabled: false,
      buttons: [{
          type: 'hour',
          count: 1,
          text: '1h'
        },
        {
          type: 'hour',
          count: 2,
          text: '2h'
        },
        {
          type: 'hour',
          count: 4,
          text: '4h'
        },
        {
          type: 'hour',
          count: 6,
          text: '6h'
        },
        {
          type: 'hour',
          count: 12,
          text: '12h'
        },
        {
          type: 'all',
          text: 'All'
        }
      ],
			             },
                 navigator: {
                    enabled: true                 
                   }, 
                  credits: {
            enabled: false,
            position: {
                align: 'right',
                x: -10,
                verticalAlign: 'bottom',
                y: -5
                 },
                 href: "https://highcharts.com",
                   text: "Highcharts"
                  },
                  ///////
                    xAxis: {
                        ordinal: false,
                        labels: {
                            style: {
                                color: "#000"
                            }
                        },
                        type: "datetime",
                        dateTimeLabelFormats: {
                            millisecond: "%H:%M:%S",
                            second: "%H:%M:%S",
                            minute: "%H:%M",
                            hour: "%H:%M",
                            day: "%e. %b",
                            week: "%e. %b",
                            month: "%b '%y",
                            year: "%Y"
                        }
                    },
                    yAxis: {
                        title: {
                            text: "Hashrate",
                            style: {
                                color: "#000"
                            },
                        },
                        labels: {
                            style: {
                                color: "#000"
                            }
                        },
                        //softMin: e.getWithDefault("model.currentHashrate") / 1000000,
                        //softMax: e.getWithDefault("model.currentHashrate") / 1000000,
                    },
                    plotLines: [{
                        value: 0,
                        width: 1,
                        color: "#808080"
                    }],
                    legend: {
                        enabled: true,
                        itemStyle:
                          {
                            color: "#000"
                          },
                    },
                    tooltip: {
                        formatter: function() {
                            return this.y > 1000000000000 ? "<b>" + this.point.d + "<b><br>Hashrate&nbsp;" + (this.y / 1000000000000).toFixed(2) + "&nbsp;TH/s</b>" : this.y > 1000000000 ? "<b>" + this.point.d + "<b><br>Hashrate&nbsp;" + (this.y / 1000000000).toFixed(2) + "&nbsp;GH/s</b>" : this.y > 1000000 ? "<b>" + this.point.d + "<b><br>Hashrate&nbsp;" + (this.y / 1000000).toFixed(2) + "&nbsp;MH/s</b>" : "<b>" + this.point.d + "<b><br>Hashrate&nbsp;<b>" + this.y.toFixed(2) + "&nbsp;H/s</b>";

                        },

                        useHTML: true
                    },
                    exporting: {
                        enabled: true
                    },
                    series: [{
                        color: "#15BD27",
                        name: "3 hours average hashrate",
                        data: function() {
                            var e, a = [];
                            if (null != t) {
                                for (e = 0; e <= t.length - 1; e += 1) {
                                    var n = 0,
                                        r = 0,
                                        l = 0;
                                    r = new Date(1e3 * t[e].x);
                                    l = r.toLocaleString();
                                    n = t[e].minerLargeHash;
                                    a.push({
                                        x: r,
                                        d: l,
                                        y: n
                                    });
                                }
                            } else {
                                a.push({
                                x: 0,
                                d: 0,
                                y: 0
                                });
                            }
                            return a;
                        }()
                    }, {
                        name: "30 minutes average hashrate",
                        color: "#E99002",
                        data: function() {
                            var e, a = [];
                            if (null != t) {
                                for (e = 0; e <= t.length - 1; e += 1) {
                                    var n = 0,
                                        r = 0,
                                        l = 0;
                                    r = new Date(1e3 * t[e].x);
                                    l = r.toLocaleString();
                                    n = t[e].minerHash;
                                    a.push({
                                        x: r,
                                        d: l,
                                        y: n
                                    });
                                }
                            } else {
                                a.push({
                                    x: 0,
                                    d: 0,
                                    y: 0
                                });
                            }
                            return a;
                        }()
              
                    }]
                };
            return a;
        }
    }),
  
  shareChart: Ember.computed("model.hashrate", {
        get() {
            var e = this,
                t = e.getWithDefault("model.shareCharts"),
                a = {
                    chart: {
                        backgroundColor: "rgba(255, 255, 255, 0.1)",
                        type: "column",
                        marginRight: 10,
                        height: 180
                       // events: {
                          /*  load: function() {
                                var series = this.series[0];
                                setInterval(function() {
                                    var x = (new Date()).getTime(),
                                        y = e.getWithDefault("model.workersOnline") / 1000000;
                                    series.addPoint([x, y], true, true);
                                }, 1090000000);
                            } */
                       // }
                    },
                    title: {
                        text: ""
                    },
                  xAxis: {
                        ordinal: false,
                        labels: {
                            style: {
                                color: "#000"
                            }
                        },
                        type: "datetime",
                        dateTimeLabelFormats: {
                            millisecond: "%H:%M:%S",
                            second: "%H:%M:%S",
                            minute: "%H:%M",
                            hour: "%H:%M",
                            day: "%e. %b",
                            week: "%e. %b",
                            month: "%b '%y",
                            year: "%Y"
                        }
                    },
                   //rangeSelector: {
                         //  selected: 1,
                         // },
                    yAxis: {
                        title: {
                            text: "Shares",
                            style: {
                                color: "#000"
                            },
                        }, 
                        labels: {
                            style: {
                                color: "#000"
                            }
                        }
                        //softMin: e.getWithDefault("model.currentHashrate") / 1000000,
                        //softMax: e.getWithDefault("model.currentHashrate") / 1000000,
                    },
                    plotOptions: {
                      series: {
                        marginleft: 0,
                       pointWidth: 10
                       //   marker: {
                                //  enabled: false
                                //   }
                        },
                      column: {
                            stacking: 'normal',
                            grouping: false
                            //shadow: false
                            //borderWidth: 0
                            }
                   },
                    plotLines: [{
                        value: 0,
                        width: 1,
                        color: "#aaaaaa"
                    }],
                    legend: {
                        enabled: true,
                        itemStyle:
                          {
                            color: "#000"
                          },
                    },
                    tooltip: {
                        formatter: function() {
                            return this.y > 1000000000000 ? "<b>" + this.point.d + "<b><br>Shares&nbsp;" + (this.y / 1000000000000) + "</b>" : this.y > 1000000000 ? "<b>" + this.point.d + "<b><br>Shares&nbsp;" + (this.y / 1000000000) + "</b>" : this.y > 1000000 ? "<b>" + this.point.d + "<b><br>Shares&nbsp;" + (this.y / 1000000) + "</b>" : "<b>" + this.point.d + "<b><br>Shares&nbsp;<b>" + this.y + "</b>";

                        },

                        useHTML: true
                    },
                   exporting: {
            enabled: false
        },
                    series: [{
                        color: "#15BD27",
                        name: "Valid share",
                        data: function() {
                            var e, a = [];
                            if (null != t) {
                                for (e = 0; e <= t.length - 1; e += 1) {
                                    var n = 0,
                                        r = 0,
                                        l = 0;
                                    r = new Date(1e3 * t[e].x);
                                    l = r.toLocaleString();
                                    n = t[e].valid;
                                    a.push({
                                        x: r,
                                        d: l,
                                        y: n
                                    });
                                }
                            } else {
                                a.push({
                                x: 0,
                                d: 0,
                                y: 0
                                });
                            }
                            return a;
                        }()
                      
                    }, {
     
                        name: "Stale share",
                        color: "#E99002",
                        data: function() {
                            var e, a = [];
                            if (null != t) {
                                for (e = 0; e <= t.length - 1; e += 1) {
                                    var n = 0,
                                        r = 0,
                                        l = 0;
                                    r = new Date(1e3 * t[e].x);
                                    l = r.toLocaleString();
                                    n = t[e].stale;
                                    a.push({
                                        x: r,
                                        d: l,
                                        y: n
                                    });
                                }
                            } else {
                                a.push({
                                    x: 0,
                                    d: 0,
                                    y: 0
                                });
                            }
                            return a;
                        }()
                      
                     /*  }, {
     
                        name: "Workers",
                        color: "#FF0000",
                        type: 'spline',
                          plotLines: [{
                       // value: 0,
                        width: 1,
                        color: "#aaaaaa"
                    }],
                        data: function() {
                            var e, a = [];
                            if (null != t) {
                                for (e = 0; e <= t.length - 1; e += 1) {
                                    var n = 0,
                                        r = 0,
                                        l = 0;
                                    r = new Date(1e3 * t[e].x);
                                    l = r.toLocaleString();
                                    n = t[e].workerOnline;
                                    a.push({
                                        x: r,
                                        d: l,
                                        y: n
                                    });
                                }
                            } else {
                                a.push({
                                    x: 0,
                                    d: 0,
                                    y: 0
                                });
                            }
                            return a;
                        }() */
                    }]
                };
            return a;
        }
    })
});
