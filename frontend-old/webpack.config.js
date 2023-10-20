const path = require("path");
const CssExtractPlugin = require('mini-css-extract-plugin');
const LiveReloadPlugin = require('webpack-livereload-plugin');
const CopyPlugin = require('copy-webpack-plugin');
const { VueLoaderPlugin } = require('vue-loader');
const VuetifyLoaderPlugin = require('vuetify-loader/lib/plugin')
const BundleAnalyzerPlugin = require('webpack-bundle-analyzer').BundleAnalyzerPlugin;
const HtmlWebpackPlugin = require('html-webpack-plugin')

const contentBase = path.join(__dirname, "/build");

const LIVE_RELOAD_PORT = 35736
const DEVELOPMENT_MODE='development'

const isDevelopment = (argv) => {
    return argv.mode === DEVELOPMENT_MODE
}

const getLiveReload = (argv) => {
    return isDevelopment(argv) ? `
        <script type="application/javascript">
            const livereloadProtocolHost = window.location.protocol + "//" + window.location.hostname;
            const scriptTag = document.createElement('script');
            scriptTag.src = livereloadProtocolHost + ":${LIVE_RELOAD_PORT}/livereload.js";
            document.head.appendChild(scriptTag);
        </script>
        ` : ""
}

module.exports = (env, argv) => {
    const currDate = isDevelopment(argv) ? "" : +new Date();
    const pluginsArray = [
        // new BundleAnalyzerPlugin({defaultSizes: "parsed"}),
        new HtmlWebpackPlugin({ // Load a custom template (lodash by default)
            currDate: currDate,
            livereload: getLiveReload(argv),
            template: './src/index.template.html',
            filename: 'index.html',
            inject: false
        }),
        new HtmlWebpackPlugin({ // Load a custom template (lodash by default)
            currDate: currDate,
            livereload: getLiveReload(argv),
            template: './src/indexBlog.template.html',
            filename: 'indexBlog.html',
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
            filename: `[name]_${currDate}.css`,
            chunkFilename: `[id]_${currDate}.css`,
            ignoreOrder: false, // Enable to remove warnings about conflicting order
        }),
    ];
    if (isDevelopment(argv)) {
        console.log("Starting LiveReloadPlugin");
        pluginsArray.push(
            new LiveReloadPlugin({
                port: LIVE_RELOAD_PORT
            })
        );
    }

    const webpackCfg = {
        entry: {
            main: "./src/main.js",
            blogMain: "./src/blogMain.js"
        },
        output: {
            path: contentBase,
            filename: `[name]_${currDate}.js`,
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
                    type: 'asset/resource',
                    generator: {
                        filename: 'fonts/[name][ext]'
                    }
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
                            },
                        },
                    ],
                },
                {
                    test: /\.less$/i,
                    use: [
                        // compiles Less to CSS
                        "style-loader",
                        "css-loader",
                        "less-loader",
                    ],
                },
            ]
        },
        plugins: pluginsArray,
    };

    if (isDevelopment(argv)) {
        // https://github.com/vuejs/vue-loader/issues/620#issuecomment-363931521
        webpackCfg.devtool = 'source-map';
    }

    return webpackCfg;
};
