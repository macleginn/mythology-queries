app.controller('MainController', ['$scope', '$http', function($scope, $http) {
	$scope.init = function() {
		$scope.hidethis = true;
		$scope.waitformap1 = true;
		$scope.map1initialised = false;
		$scope.map1markers = [];
		$scope.waitformap2 = true;
		$scope.fetchData();
		$scope.ns = [];
		for (var i = 1; i <= 30; i++) {
			$scope.ns.push(i);
		}
	};
	// TODO: replace with text input with autocomplete options
	$scope.fetchData = function() {
		$http.get('http://23.254.167.151:8080/fetchMotifList').then(function(response) {
			$scope.selectedMotif = "Choose the motif";
			$scope.hideuntilcompare = true;
			$scope.hidethis = false;
			$scope.queryresults = response.data;
		}, function(response) {
			$scope.queryresults = "Data retrieval error";
		});
	};
	$scope.showSelectedMotif = function() {
		// Initialise the map for the selected motif
		if (!$scope.map1initialised) {
			styles = [{"featureType":"landscape","stylers":[{"saturation":-100},{"lightness":65},{"visibility":"on"}]},{"featureType":"poi","stylers":[{"saturation":-100},{"lightness":51},{"visibility":"simplified"}]},{"featureType":"road.highway","stylers":[{"saturation":-100},{"visibility":"simplified"}]},{"featureType":"road.arterial","stylers":[{"saturation":-100},{"lightness":30},{"visibility":"on"}]},{"featureType":"road.local","stylers":[{"saturation":-100},{"lightness":40},{"visibility":"on"}]},{"featureType":"transit","stylers":[{"saturation":-100},{"visibility":"simplified"}]},{"featureType":"administrative.province","stylers":[{"visibility":"off"}]},{"featureType":"water","elementType":"labels","stylers":[{"visibility":"on"},{"lightness":-25},{"saturation":-100}]},{"featureType":"water","elementType":"geometry","stylers":[{"hue":"#ffff00"},{"lightness":-25},{"saturation":-97}]}];
			var center = new google.maps.LatLng(39.149908, 22.153219);
			var mapOptions = {
				zoom: 1,
				center: center,
				mapTypeId: google.maps.MapTypeId.TERRAIN
			};
			$scope.map1 = new google.maps.Map(document.getElementById('map1Div'), mapOptions);
			$scope.map1.setOptions({styles: styles});
			$scope.waitformap1 = false;
			$scope.map1initialised = true;
		}
		for (var i = 0; i < $scope.map1markers.length; i++) {
			$scope.map1markers[i].setMap(null);
			$scope.map1markers[i] = null;
		}
		$scope.map1markers = [];
		var urlstring = 'http://23.254.167.151:8080/fetchMotifDistr?code=' + encodeURIComponent($scope.selectedMotif);
		$http.get(urlstring).then(function(response) {
			// console.log(response);
			for (var i = 0; i < response.data.length; i++) {
				var point = response.data[i];
				var latLng = new google.maps.LatLng(point["Latitude"], point["Longitude"]);
				var marker = new google.maps.Marker({
					position: latLng,
					map: $scope.map1,
					title: point["Name"],
					icon: {
						path: google.maps.SymbolPath.CIRCLE,
						fillColor: "yellow",
						fillOpacity: 1,
						scale: 4,
						strokeWeight: 1,
						strokeColor: "black"
					}
				});
				$scope.map1markers.push(marker);
			}
		}, function(response) {
			console.log("Data retrieval error");
		});
	};
	$scope.sendMotifQuery = function() {
		if (!($scope.nmotifs === undefined)) {
			var urlstring = 'http://23.254.167.151:8080/motifQuery?code=' + encodeURIComponent($scope.selectedMotif) + '&num=' + $scope.nmotifs;
			$http.get(urlstring).then(function(response) {
				$scope.hideuntilcompare = true;
				$scope.neighMotifs = response.data;
			}, function(response) {
				$scope.neighMotifs = "Data retrieval error";
			});
		}
	};
	// TODO!!!
	$scope.showOnTheMap = function(code) {
		alert(code);
	}
}]);