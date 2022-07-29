import Ember from 'ember';

export default Ember.Controller.extend({
  applicationController: Ember.inject.controller('application'),
  config: Ember.computed.reads('applicationController.config'),
  settings: Ember.computed.reads('applicationController.model.settings'),

  BlockUnlockDepth: Ember.computed('settings', {
    get() {
      var depth = this.get('settings.BlockUnlockDepth');
      if (depth) {
        return depth;
      }
      return this.get('config').BlockUnlockDepth;
    }
  }),

  chartOptions: Ember.computed("model.luckCharts", {
        get() {
            var e = this,
                t = e.getWithDefault("model.luckCharts"),
                a = {
                    colors: ['#f45b5b', '#8085e9', '#8d4654', '#7798BF', '#aaeeee',
                            '#ff0066', '#eeaaee', '#55BF3B', '#DF5353', '#7798BF', '#aaeeee'],
                    chart: {
                        backgroundColor: "rgba(255, 255, 255, 0.1)",
                        marginRight: 10,
                        height: 200,
                        events: {
                            load: function() {
                                var series = this.series[0];
                                setInterval(function() {
                                    var x = (new Date()).getTime(),
                                        y = e.getWithDefault("model.luckCharts.difficulty");
                                    series.addPoint([x, y], true, true);
                                }, 1090000000);
                            }
                        }
                    },
                    title: {
                        text: ""
                    },
                    xAxis: {
                        labels: {
                            style: {
                                color: '#6e6e70'
                            }
                        },
                        ordinal: false,
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
                        labels: {
                            style: {
                                color: '#6e6e70'
                            }
                        },
                        title: {
                            text: "shares and difficulty",
                            style: {
                                color: 'black',
                                fontSize: '16px',
                                fontWeight: 'bold'
                            }
                        },
                        softMax: 100,
                    },
                    plotLines: [{
                        value: 0,
                        width: 1,
                        color: "#808080"
                    }],
                    plotOptions: {
                        series: {
                            shadow: true
                        },
                        candlestick: {
                            lineColor: '#404048'
                        },
                        map: {
                            shadow: false
                        }
                    },
                    legend: {
                        enabled: true
                    },
                    tooltip: {
                        formatter: function() {
                            var ss = this.y > 1000000000000 ? "<b>" + (this.y / 1000000000000).toFixed(2) + "&nbsp;TH</b>" : this.y > 1000000000 ? "<b>" + (this.y / 1000000000).toFixed(2) + "&nbsp;GH</b>" : this.y > 1000000 ? "<b>" + (this.y / 1000000).toFixed(2) + "&nbsp;MH</b>" : this.y > 1000 ? "<b>" + (this.y / 1000).toFixed(2) + "&nbsp;KH</b>" : "<b>" + this.y.toFixed(2) + "&nbsp;H</b>";
                            return ss + "<br/><b>Number:&nbsp;" + this.point.h + "</b><br/><b>" + this.point.d + "</b><br/><b>Reward:&nbsp;" + (this.point.w/1000000000000000000).toFixed(8) + e.get('config.Unit') + " </b><br/><b>Variance:&nbsp;" + (this.point.s*100).toFixed(2)+ "%</b>";
                        },

                        useHTML: true
                    },
                    exporting: {
                        enabled: false
                    },
                    series: [{
                        step: 'center',
                        color: "#E99002",
                        name: "difficulty",
                        data: function() {
                            var e, a = [];
                            if (null != t) {
                                for (e = 0; e <= t.length - 1; e += 1) {
                                    var n = 0,
                                        r = 0,
                                        l = 0;
                                        r = new Date(1e3 * t[e].x);
                                        l = r.toLocaleString();
                                        n = t[e].difficulty;
                                        a.push({
                                            x: r,
                                            d: l,
                                            h: t[e].height,
                                            w: t[e].reward,
                                            s: t[e].sharesDiff,
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
                        step: 'center',
                        name: "shares",
                        data: function() {
                            var e, a = [];
                            if (null != t) {
                                for (e = 0; e <= t.length - 1; e += 1) {
                                    var n = 0,
                                        r = 0,
                                        l = 0;
                                        r = new Date(1e3 * t[e].x);
                                        l = r.toLocaleString();
                                        n = t[e].shares;
                                        a.push({
                                            x: r,
                                            d: l,
                                            h: t[e].height,
                                            w: t[e].reward,
                                            s: t[e].sharesDiff,
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
    })

});
