var app = angular.module("queryApp", []);
app.filter('formatList', function() {
    return function(arr) {
    	if (arr === undefined) {
    		arr = [];
    	}
        return arr.join(", ");
    };
});
app.filter('round', function() {
	return function(num) {
		return Math.round(num);
	}
});