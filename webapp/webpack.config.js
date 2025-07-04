const path = require('path');

module.exports = {
    entry: './src/index.tsx',
    mode: 'production',
    resolve: {
        extensions: ['.ts', '.tsx', '.js', '.jsx'],
        alias: {
            'mattermost-redux': path.resolve(__dirname, '../node_modules/mattermost-redux'),
            'types/mattermost-webapp': path.resolve(__dirname, 'src/types/mattermost-webapp.d.ts'),
        },
    },
    module: {
        rules: [
            {
                test: /\.(ts|tsx)$/,
                use: 'ts-loader',
                exclude: /node_modules/,
            },
            {
                test: /\.css$/,
                use: ['style-loader', 'css-loader'],
            },
        ],
    },
    externals: {
        'react': 'React',
        'react-dom': 'ReactDOM',
        'redux': 'Redux',
        'react-redux': 'ReactRedux',
        'prop-types': 'PropTypes',
        'mattermost-redux/client': 'window.MattermostRedux?.Client4',
        'mattermost-redux/types/store': 'window.MattermostRedux?.Types',
    },
    output: {
        path: path.resolve(__dirname, 'dist'),
        filename: 'main.js',
        library: {
            name: 'window["plugins"]["com.mattermost.exchange-plugin"]',
            type: 'assign-properties'
        },
    },
    devtool: 'source-map',
}; 