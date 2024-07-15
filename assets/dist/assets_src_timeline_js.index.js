"use strict";
/*
 * ATTENTION: The "eval" devtool has been used (maybe by default in mode: "development").
 * This devtool is neither made for production nor for readable output files.
 * It uses "eval()" calls to create a separate source file in the browser devtools.
 * If you are trying to read the output file, select a different devtool (https://webpack.js.org/configuration/devtool/)
 * or disable the default devtool with "devtool: false".
 * If you are looking for production-ready output files, see mode: "production" (https://webpack.js.org/configuration/mode/).
 */
(self["webpackChunkbeautybargains"] = self["webpackChunkbeautybargains"] || []).push([["assets_src_timeline_js"],{

/***/ "./assets/src/timeline.js":
/*!********************************!*\
  !*** ./assets/src/timeline.js ***!
  \********************************/
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

eval("__webpack_require__.r(__webpack_exports__);\n/* harmony export */ __webpack_require__.d(__webpack_exports__, {\n/* harmony export */   timeline: () => (/* binding */ timeline)\n/* harmony export */ });\n/* harmony import */ var chart_js_auto__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! chart.js/auto */ \"./node_modules/chart.js/auto/auto.js\");\n\r\nfunction timeline() {\r\n    const _productData = document.querySelector(\"#productdata\")?.textContent\r\n    if(!_productData) {\r\n        console.log(\"no product data found\");\r\n        return\r\n    }\r\n\r\n    let productData\r\n    try {\r\n        productData = JSON.parse(_productData)\r\n        if (typeof productData === \"string\") {\r\n            productData = JSON.parse(productData)\r\n        }\r\n    } catch(err) {\r\n        console.log(err)\r\n        return\r\n    }\r\n\r\n    if(!productData) {\r\n        console.log(\"no product data\")\r\n        return\r\n    }\r\n\r\n    console.log(typeof productData);\r\n    console.log(productData.dates);\r\n    console.log(productData.prices);\r\n\r\n    const ctx = document.getElementById('price-chart');\r\n    if (!ctx) {\r\n        console.log(\"no chart element detected\")\r\n        return\r\n    }\r\n\r\n    new chart_js_auto__WEBPACK_IMPORTED_MODULE_0__[\"default\"](ctx, {\r\n        type: 'line',\r\n        data: {\r\n            labels: productData.dates,\r\n            datasets: [{\r\n                label: 'Recorded Prices',\r\n                data: productData.prices,\r\n                borderWidth: 1\r\n            }]\r\n        },\r\n        options: {\r\n            scales: {\r\n                y: {\r\n                    beginAtZero: true\r\n                }\r\n            }\r\n        }\r\n    });\r\n\r\n}\r\n\n\n//# sourceURL=webpack://beautybargains/./assets/src/timeline.js?");

/***/ })

}]);