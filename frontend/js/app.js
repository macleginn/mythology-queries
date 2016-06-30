var app = angular.module("queryApp", []);
app.filter('formatList', function() {
    return function(arr) {
        return arr.join(", ");
    };
});
app.filter('round', function() {
	return function(num) {
		return Math.round(num);
	}
});