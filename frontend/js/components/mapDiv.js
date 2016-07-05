template = `<p><strong>{{ $ctrl.divid }}</strong></p>
<div id="{{ $ctrl.divid }}" class="mapDiv"></div>`

MapDivController = function() {
};

app.component('mapDiv', {
	template: template,
	bindings: {
		divid: '@',
	},
	controller: MapDivController,
});