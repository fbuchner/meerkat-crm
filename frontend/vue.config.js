const webpack = require('webpack');

module.exports = {
  transpileDependencies: true,
  configureWebpack: {
    plugins: [
      new webpack.DefinePlugin({
        __VUE_PROD_DEVTOOLS__: JSON.stringify(false),
        __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: JSON.stringify(false),
      }),
    ],
  },
  pages: {
    index: {
      entry: 'src/main.js',
      template: 'public/index.html', 
      title: 'perema | personal CRM system',
    },
  },
};
