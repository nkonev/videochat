const path = require("path");
const CssExtractPlugin = require('mini-css-extract-plugin');
const LiveReloadPlugin = require('webpack-livereload-plugin');
const CopyPlugin = require('copy-webpack-plugin');

const contentStatic = path.join(__dirname, "../frontend-nginx");
const contentBase = path.join(contentStatic, "/public");

module.exports = (env, argv) => {
    const pluginsArray = [
        new CopyPlugin([
            { from: './public', to: contentStatic },
        ]),
        new CssExtractPlugin({
            // Options similar to the same options in webpackOptions.output
            // all options are optional
            filename: '[name].css',
            chunkFilename: '[id].css',
            ignoreOrder: false, // Enable to remove warnings about conflicting order
        }),
    ];
    if (argv.mode === 'development') {
        console.log("Starting LiveReloadPlugin");
        pluginsArray.push(
            new LiveReloadPlugin({
                appendScriptTag: true,
                port: 35736
            })
        );
    }

    return {
        entry: "./src/main.js",
        output: {
            path: contentBase,
            filename: "main.js"
        },
        module: {
            rules: [
                {
                    test: /\.js$/,
                    exclude: /node_modules/,
                    use: {
                        loader: "babel-loader"
                    },
                },
                {
                    test: /\.css$/,
                    use: [
                        {
                            loader: CssExtractPlugin.loader,
                            options: {
                                hot: process.env.NODE_ENV === 'development',
                            },
                        },
                        'css-loader',
                    ]
                },
                {
                    test: /\.(svg)$/,
                    exclude: /fonts/, /* dont want svg fonts from fonts folder to be included */
                    use: [
                        {
                            loader: 'svg-url-loader',
                            options: {
                                noquotes: true,
                            },
                        },
                    ],
                },
                {
                    test: /\.(ttf|eot|woff|woff2)$/,
                    use: [
                        {
                            loader: 'url-loader',
                            options: {
                                name: '[path][name].[ext]',
                                limit: '4096'
                            }
                        }
                    ],
                },
            ]
        },
        plugins: pluginsArray,
    }
};