const path = require('path');

module.exports = {
    mode: "production",
    entry: './assets/src/index.js',
    output: {
        filename: 'index.js',
        path: path.resolve(__dirname, 'assets/dist'),
    },
};
