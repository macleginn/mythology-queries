app.controller('MainController', ['$scope', '$http', function($scope, $http) {
	$scope.init = function() {
		$scope.hidethis = true;
		$scope.hideuntildata = true;
		$scope.fetchData();
		$scope.ns = [];
		for (var i = 1; i <= 30; i++) {
			$scope.ns.push(i);
		}
	};
	// TODO: replace with text input with autocomplete options
	$scope.fetchData = function() {
		$http.get('http://localhost:8080/fetchTraditionDict').then(function(response) {
			$scope.selectedTradition = "Choose the tradition";
			$scope.hidethis = false;
			$scope.queryresults = response.data;
		}, function(response) {
			$scope.queryresults = "Data retrieval error";
		})
	};
	$scope.toggleChange = function() {
		$scope.toggledbychange = $scope.selectedTradition.Name;
	};
	$scope.sendTraditionQuery = function() {
		var urlstring = 'http://localhost:8080/traditionQuery?code=' + encodeURIComponent($scope.selectedTradition.Name) + '&num=' + $scope.ntrads;
		$http.get(urlstring).then(function(response) {
			$scope.hideuntildata = false;
			$scope.neighTrads = response.data;
		}, function(response) {
			$scope.neighTrads = "Data retrieval error";
		})
	}
}]);