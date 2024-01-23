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

eval("__webpack_require__.r(__webpack_exports__);\n/* harmony export */ __webpack_require__.d(__webpack_exports__, {\n/* harmony export */   timeline: () => (/* binding */ timeline)\n/* harmony export */ });\n/* harmony import */ var chart_js_auto__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! chart.js/auto */ \"./node_modules/chart.js/auto/auto.js\");\n\nfunction timeline() {\n    const _productData = document.querySelector(\"#productdata\")?.textContent\n    if(!_productData) {\n        console.log(\"no product data found\");\n        return\n    }\n\n    let productData\n    try {\n        productData = JSON.parse(_productData)\n        if (typeof productData === \"string\") {\n            productData = JSON.parse(productData)\n        }\n    } catch(err) {\n        console.log(err)\n        return\n    }\n\n    if(!productData) {\n        console.log(\"no product data\")\n        return\n    }\n\n    console.log(typeof productData);\n    console.log(productData.dates);\n    console.log(productData.prices);\n\n    const ctx = document.getElementById('price-chart');\n    if (!ctx) {\n        console.log(\"no chart element detected\")\n        return\n    }\n\n    new chart_js_auto__WEBPACK_IMPORTED_MODULE_0__[\"default\"](ctx, {\n        type: 'line',\n        data: {\n            labels: productData.dates,\n            datasets: [{\n                label: 'Recorded Prices',\n                data: productData.prices,\n                borderWidth: 1\n            }]\n        },\n        options: {\n            scales: {\n                y: {\n                    beginAtZero: true\n                }\n            }\n        }\n    });\n\n}\n\n\n//# sourceURL=webpack://beautybargains/./assets/src/timeline.js?");

/***/ })

}]);