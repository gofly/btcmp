function BtCmp(opt) {
    this.$matrix = {};
    this.wsHost = opt.wsHost;
    this.pairType = opt.pairType;
    this.$priceList = opt.priceListEl;
    this.tplStr = opt.tplStr;
    this.thresholds = {};
    this.notifyEl = opt.notifyEl;
    this.musicNotify = false;
    this.LoadSettings();
}

BtCmp.prototype.DialWS = function() {
    var that = this;
    var ws = new WebSocket(this.wsHost + '/ws/pairs/' + this.pairType.toLowerCase());
    ws.onopen = function (evt) {
        console.log('ws connected');
    };
    ws.onclose = function (evt) {
        console.log('ws connection closed');
	setTimeout(function(){
            window.location.reload(true);
        }, 3000);
    };
    ws.onmessage = function (evt) {
        var data = JSON.parse(evt.data);
        switch (data.Method) {
        case 'Tickers.Update':
            that.RenderList(data.Data);
            that.Alert()
            break;
        case 'Thresholds.Reload':
            that.LoadSettings();
            break;
        case 'Ping':
            ws.send(JSON.stringify({Method: 'Pong'}));
        }
    };
    ws.onerror = function (evt) {
        console.log('ws connection error');
	setTimeout(function(){
            window.location.reload(true);
        }, 3000);
    };
}

BtCmp.prototype.LoadPriceData = function () {
    var that = this;
    $.getJSON('/api/pairs/' + this.pairType.toLowerCase())
    .done(function(data, textStatus, jqXHR) {
        if(data.length > 0) {
            that.$priceList.html('');
        }
        for(var i = 0; i < data.length; i++) {
            that.RenderList(data[i]);
            that.Alert()
        }
        that.DialWS();
    })
    .fail(function(jqXHR, textStatus) {
        if(jqXHR.status === 401){
            window.location.href = '/login';
        } else{
            console.log(jqXHR.status);
        }
    });
};

BtCmp.prototype.LoadSettings = function(){
    var that = this;
    $.getJSON('/api/settings/' + this.pairType.toLowerCase())
    .done(function(data, textStatus, jqXHR) {
        var i;
        if(data){
            that.musicNotify = data.MusicNotify;
            if(data.Thresholds){
                for(i = 0; i < data.Thresholds.length; i++){
                    that.thresholds[data.Thresholds[i].PairName] = data.Thresholds[i].Value;
                }
            }
        }
    })
    .fail(function(jqXHR, textStatus) {
        console.log(jqXHR.status);
    });
}

BtCmp.prototype.LineCompare = function(pair){
    var vendor, el, value1, value2, threshold, el1, el2;
    if(!(pair in this.thresholds)){
        return;
    }
    threshold = this.thresholds[pair];
    for(vendor in this.$matrix[pair]){
        this.$matrix[pair][vendor].removeClass('table-danger');
    }
    for(vendor1 in this.$matrix[pair]){
        el1 = this.$matrix[pair][vendor1];
        value1 = parseFloat(el1.html());
        for(vendor2 in this.$matrix[pair]){
            if(vendor1 === vendor2 || vendor1 === 'Gateio' || vendor2 === 'Gateio' || vendor1 === 'Okex' || vendor2 === 'Okex'){
                continue;
            }
            el2 = this.$matrix[pair][vendor2];
            value2 = parseFloat(el2.html());
            if(threshold > 0 && value2 != 0 && ((value1 - value2) / value2) > threshold/100){
                el1.addClass('table-danger');
                el2.addClass('table-danger');
                console.log(((value1 - value2) / value2),'>', threshold/100);
            }
        }
    }
};

BtCmp.prototype.Alert = function(){
    if(this.musicNotify && this.$priceList.find('.table-danger').length > 0){
        if(this.notifyEl[0].ended || this.notifyEl[0].paused || this.notifyEl.data('flag') === 0){
            this.notifyEl.data('flag', 1);
            this.notifyEl[0].currentTime = 0;
            this.notifyEl[0].play();
        }
    } else{
        this.notifyEl[0].pause();
    }
};

BtCmp.prototype.RenderList = function(data) {
    var ticker, pair, vendor, $item, $items = [];
    var toFixed = function(n, t) {
        var i = String(Math.round(n * Math.pow(10, t))), r, u;
        if (0 < t) {
            if (r = i.length,
            r < t)
                for (u = 0; u < t - r; u++)
                    i = "0" + i;
            return r = i.substring(0, i.length - t),
            "" === r && (r = 0),
            r + "." + i.substring(i.length - t, i.length)
        }
        return String(i)
    };
    for(pair in data) {
        ticker = {PairName: pair};
        for (vendor in data[pair]) {
            ticker[vendor] = toFixed(data[pair][vendor].Last, 8);
        }
        if(this.$matrix[pair] === undefined){
            this.$matrix[pair] = {};
            $item = $(this.tplStr);
            for(vendor in ticker) {
                this.$matrix[pair][vendor] = $item.find('.' + vendor);
            }
            $items.push($item);
        }
        for(vendor in ticker){
		el = this.$matrix[pair][vendor];
		//el.css({'background-color': 'transparent'});
                //    el.addClass('updated');
		el.html(ticker[vendor]);
		//setTimeout(function(){
		//el.removeClass('updated');
		//el.css({'background-color': '#FFEB3B'});
                //},0);
        }
        this.LineCompare(pair);
    }
    this.$priceList.append($items);
};

function BtSettings(el) {
    this.$el = el;
}

BtSettings.prototype.Bind = function(){
    var that = this;
    this.$el.delegate('.restart', 'click', function(e){
        that.restartProcess(e);
    });
    this.$el.delegate('.add', 'click', function(e){
        that.addThreshold(e);
    });
    this.$el.delegate('.delete', 'click', this.deleteThreshold);
    this.$el.delegate('.save', 'click', function(e){
        that.saveSettings(e);
    });
};

BtSettings.prototype.getLineValue = function($tr){
    var $pairEl = $tr.find('input[name="pair"]'),
        pairType = $tr.data('type'),
        baseName = $pairEl.val(),
        $thresholdEl = $tr.find('input[name="threshold"]');
    return {
        pairType: pairType,
        pairEl: $pairEl,
        baseName: baseName,
        thresholdEl: $thresholdEl,
        pairName: pairType + '-' + baseName,
        threshold: parseFloat($thresholdEl.val())
    }
};
BtSettings.prototype.addThreshold = function(e){
    var $tr = $(e.currentTarget).parents('tr'),
        $newTr = $tr.clone(),
        val = this.getLineValue($tr);
    if (val.baseName === '') {
        alert('币种不能为空');
        return
    }
    if (val.threshold === ''){
        alert('阈值不能为空');
    }
    $newTr.find('.add').hide();
    $newTr.find('.delete').show();
    $tr.before($newTr);
    val.pairEl.val('');
    val.thresholdEl.val('');
};

BtSettings.prototype.deleteThreshold = function(e){
    var $tr = $(e.currentTarget).parents('tr');
    $tr.remove();
};

BtSettings.prototype.saveSettings = function(e) {
    var $notifyCheck = this.$el.find('input[name="music-notify"]'),
        $trs = this.$el.find('tbody tr'), i, $tr, val, thresholds = [],
        musicNotify = $notifyCheck.prop('checked');
    for(i = 0; i < $trs.length; i++) {
        $tr = $($trs[i]);
        val = this.getLineValue($tr);
        if(val.baseName === ''){
            continue;
        }
        thresholds.push({
            PairName: val.pairName,
            Value: parseFloat(val.threshold)
        });
    }
    $.ajax({
        type: 'POST',
        url: '/api/settings',
        data: JSON.stringify({
            MusicNotify: musicNotify,
            Thresholds: thresholds
        }),
        contentType: 'application/json; charset=utf-8',
        dataType: 'json'
    })
    .done(function(data, textStatus, jqXHR) {
        alert('保存成功');
        window.location.reload(true);
    })
    .fail(function(jqXHR, textStatus) {
        console.log(jqXHR);
    });
};

BtSettings.prototype.restartProcess = function(e) {
    var process = $(e.currentTarget).data('process');
    $.post('/api/restart', {process: process})
    .done(function(data, textStatus, jqXHR) {
        alert('重启成功，PID: '+data.Pid);
        window.location.reload(true);
    })
    .fail(function(jqXHR, textStatus) {
       alert('重启出错: ' + textStatus);
    });
};