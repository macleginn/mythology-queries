app.controller('MainController', ['$scope', '$http', function($scope, $http) {
	$scope.init = function() {
		$scope.hidethis = true;
		$scope.hideuntildata = true;
		$scope.fetchData();
		$scope.ns = [];
		for (var i = 1; i <= 30; i++) {
			$scope.ns.push(i);
		}
		$scope.comparisonResults = "";
	};
	// TODO: replace with text input with autocomplete options
	$scope.fetchData = function() {
		$http.get('http://localhost:8080/fetchTraditionDict').then(function(response) {
			$scope.selectedTradition = "Choose the tradition";
			$scope.hideuntilcompare = true;
			$scope.hidethis = false;
			$scope.queryresults = response.data;
		}, function(response) {
			$scope.queryresults = "Data retrieval error";
		});
	};
	$scope.toggleChange = function() {
		$scope.toggledbychange = $scope.selectedTradition.Name;
	};
	$scope.sendTraditionQuery = function() {
		var urlstring = 'http://localhost:8080/traditionQuery?code=' + encodeURIComponent($scope.selectedTradition.Name) + '&num=' + $scope.ntrads;
		$http.get(urlstring).then(function(response) {
			$scope.hideuntilcompare = true;
			$scope.neighTrads = response.data;
		}, function(response) {
			$scope.neighTrads = "Data retrieval error";
		});
	};
	$scope.compareWith = function(otherTradition) {
		$scope.otherTradition = otherTradition;
		var urlstring = 'http://localhost:8080/compareTraditions?trad1=' + encodeURIComponent($scope.selectedTradition.Name) + '&trad2=' + encodeURIComponent(otherTradition);
		$http.get(urlstring).then(function(response) {
			$scope.hideuntilcompare = false;
			$scope.traditionComparisonData = response.data;

		}, function(response) {
			$scope.traditionComparisonData = "Data retrieval error";
		});
	};
}]);