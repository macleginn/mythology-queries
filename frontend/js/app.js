var app = angular.module("queryApp", []);
app.filter('formatList', function() {
    return function(arr) {
        return arr.join(", ");
    };
});