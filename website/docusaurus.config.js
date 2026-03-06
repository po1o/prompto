export default {
  title: 'Prompto',
  tagline: 'The most customizable and fastest prompt engine for any shell.',
  url: 'https://prompto.dev',
  baseUrl: '/',
  favicon: 'img/favicons.svg',
  organizationName: 'jandedobbeleer',
  projectName: 'prompto',
  onBrokenLinks: 'ignore',
  plugins: [
    './plugins/appinsights'
  ],
  stylesheets: [
    "https://rsms.me/inter/inter.css",
    "https://fonts.googleapis.com/css2?family=Fira+Code&display=swap"
  ],
  themeConfig: {
    colorMode: {
      defaultMode: 'light',
      disableSwitch: false,
      respectPrefersColorScheme: true,
    },
    prism: {
      additionalLanguages: ['powershell', 'lua', 'jsstacktrace', 'toml'],
    },
    docs: {
        sidebar: {
          hideable: true,
        },
    },
    navbar: {
      title: 'Prompto',
      logo: {
        alt: 'Prompto Logo',
        src: 'img/logo-dark.svg',
        srcDark: 'img/logo-light.svg',
      },
      items: [
        {
          to: 'docs',
          activeBasePath: 'docs',
          label: 'Docs',
          position: 'left',
        },
        {
          to: 'blog',
          label: 'Blog',
          position: 'left'
        },
        {
          href: 'https://github.com/sponsors/JanDeDobbeleer',
          label: 'Sponsor',
          position: 'left',
        },
        {
          href: 'https://swag.prompto.dev',
          label: 'Swag',
          position: 'left',
        },
        {
          href: 'https://github.com/po1o/prompto',
          className: 'header-github-link',
          'aria-label': 'GitHub repository',
          position: 'right',
        },
        {
          href: 'https://discord.gg/n7E3DkXssv',
          className: 'header-discord-link',
          'aria-label': 'Discord',
          position: 'right',
        },
        {
          href: 'https://staging.bsky.app/profile/prompto.dev',
          className: 'header-bluesky-link',
          'aria-label': 'Bluesky',
          position: 'right',
        }
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'How to',
          items: [
            {
              label: 'Getting started',
              to: 'docs/',
            },
            {
              label: 'Contributing',
              to: 'docs/contributing/started',
            },
          ],
        },
        {
          title: 'Social',
          items: [
            {
              label: 'GitHub',
              href: 'https://github.com/po1o/prompto',
            },
            {
              label: 'Discord',
              href: 'https://discord.gg/n7E3DkXssv',
            },
            {
              label: 'Bluesky',
              href: 'https://staging.bsky.app/profile/prompto.dev',
            }
          ],
        },
        {
          title: 'Links',
          items: [
            {
              label: 'Sponsor',
              href: 'https://github.com/sponsors/JanDeDobbeleer',
            },
            {
              label: 'Product spotlight',
              href: 'https://buy.polar.sh/polar_cl_qnmZxboq1IDUJo03mk2Jue6ktqZrCXElnzH2s2xbV2R',
            },
            {
              label: 'Docusaurus',
              href: 'https://github.com/facebook/docusaurus',
            },
            {
              label: 'Privacy',
              href: '/privacy',
            },
          ],
        },
                {
          title: 'Sponsors',
          items: [
            {
              label: 'Merge Conflict',
              href: 'https://www.mergeconflict.fm/',
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} <a href='https://github.com/sponsors/JanDeDobbeleer' target='_blank'>Jan De Dobbeleer</a> and <a href='/docs/contributors'>contributors</a>.`,
    },
    announcementBar: {
      id: 'support_us',
      content:
        'If you\'re enjoying Prompto, consider becoming a <a target="_blank" rel="noopener noreferrer" href="https://github.com/sponsors/JanDeDobbeleer">sponsor</a> to keep the project going strong 💪',
      backgroundColor: '#2c7ae0',
      textColor: '#ffffff',
      isCloseable: false,
    },
    appInsights: {
      instrumentationKey: '51741aa7-e087-4e80-b7b0-0863d467462a',
    },
    algolia: {
      appId: 'XIR4RB3TM1',
      apiKey: '15c5f4340520612ed98fe21d15882029',
      indexName: 'prompto',
    },
  },
  presets: [
    [
      '@docusaurus/preset-classic',
      {
        docs: {
          sidebarPath: './sidebars.js',
          editUrl: 'https://github.com/po1o/prompto/edit/main/website/',
        },
        theme: {
          customCss: [
            './src/css/prism-rose-pine-moon.css',
            './src/css/custom.css'
          ],
        },
        blog: {
          onInlineAuthors: 'ignore'
        },
      },
    ],
  ],
};
