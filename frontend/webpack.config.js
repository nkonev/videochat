const path = require("path");
const CssExtractPlugin = require('mini-css-extract-plugin');
const LiveReloadPlugin = require('webpack-livereload-plugin');
const CopyPlugin = require('copy-webpack-plugin');
const { VueLoaderPlugin } = require('vue-loader');
const VuetifyLoaderPlugin = require('vuetify-loader/lib/plugin')
const BundleAnalyzerPlugin = require('webpack-bundle-analyzer').BundleAnalyzerPlugin;
const HtmlWebpackPlugin = require('html-webpack-plugin')

const contentStaticDest = path.join(__dirname, "../frontend-nginx");
const contentBase = path.join(contentStaticDest, "/build");

const LIVE_RELOAD_PORT = 35736
const DEVELOPMENT_MODE='development'

module.exports = (env, argv) => {
    const pluginsArray = [
        // new BundleAnalyzerPlugin({defaultSizes: "parsed"}),
        new HtmlWebpackPlugin({
            livereload: argv.mode === DEVELOPMENT_MODE ? `<script src="http://localhost:${LIVE_RELOAD_PORT}/livereload.js"></script>` : "",
            // Load a custom template (lodash by default)
            template: './src/index.template.html',
            inject: false
        }),
        new CopyPlugin({patterns: [
            { from: './static', to: contentBase },
        ]}),
        new VueLoaderPlugin(),
        new VuetifyLoaderPlugin(),
        new CssExtractPlugin({
            // Options similar to the same options in webpackOptions.output
            // all options are optional
            filename: '[name].css',
            chunkFilename: '[id].css',
            ignoreOrder: false, // Enable to remove warnings about conflicting order
        }),
    ];
    if (argv.mode === DEVELOPMENT_MODE) {
        console.log("Starting LiveReloadPlugin");
        pluginsArray.push(
            new LiveReloadPlugin({
                port: LIVE_RELOAD_PORT
            })
        );
    }

    const webpackCfg = {
        entry: "./src/main.js",
        output: {
            path: contentBase,
            filename: "[name].js",
        },
        resolve: {
            alias: {
                'vue$': path.resolve(path.join(__dirname, 'node_modules', 'vue/dist/vue.runtime.esm.js')), // it's important, else you will get "You are using the runtime-only build of Vue where the template compiler is not available. Either pre-compile the templates into render functions, or use the compiler-included build."
                '@': path.resolve(__dirname, 'src')
            },
            extensions: ['.js', '.vue']
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
                    use: [ // https://vue-loader.vuejs.org/ru/guide/extract-css.html#webpack-4
                        CssExtractPlugin.loader,
                        "css-loader",
                    ]
                },
                {
                    test: /\.styl|stylus$/,
                    use: [
                        CssExtractPlugin.loader,
                        "css-loader",
                        'stylus-loader'
                    ]
                },
                {
                    test: /\.(ttf|eot|woff|woff2|svg)$/,
                    use: [
                        {
                            loader: 'file-loader',
                            options: {
                                name: '[name].[ext]',
                                outputPath: 'fonts/'
                            }
                        }
                    ],
                    type: 'javascript/auto'
                },
                {
                    test: /\.vue$/,
                    loader: 'vue-loader',
                    options: {
                        extractCSS: true
                    }
                },
                {
                    test: /\.s(c|a)ss$/,
                    use: [
                        CssExtractPlugin.loader,
                        'css-loader',
                        {
                            loader: 'sass-loader',
                            // Requires sass-loader@^8.0.0
                            options: {
                                implementation: require('sass'),
                                sassOptions: {
                                    fiber: require('fibers'),
                                    indentedSyntax: true // optional
                                },
                            },
                        },
                    ],
                },
            ]
        },
        plugins: pluginsArray,
    };

    if (argv.mode === DEVELOPMENT_MODE) {
        // https://github.com/vuejs/vue-loader/issues/620#issuecomment-363931521
        webpackCfg.devtool = 'source-map';
    }

    return webpackCfg;
};