const presets = [
  ['@babel/preset-env', {
    targets: 'last 1 version',
    shippedProposals: true
  }],
  ['@babel/preset-react', {
    useBuiltIns: true
  }]
];

const plugins = [
  ['babel-plugin-styled-components', {
    minify: true,
    pure: true
  }],
  ["@babel/plugin-transform-runtime",
    {
      "regenerator": true
    }
  ]
];

module.exports = { presets, plugins };
