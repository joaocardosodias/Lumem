// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	redirects: {
		'/': '/readme',
	},
	integrations: [
		starlight({
		title: 'Documentação',
			logo: {
				light: './src/assets/logo.png',
				dark: './src/assets/logo-white.png',
			},
			favicon: '/favicon.png',
			customCss: [
				'./src/styles/custom.css',
			],
			social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/withastro/starlight' }],
			sidebar: [
				{ label: 'Solução', slug: 'readme' },
				{
					label: 'Sprints',
					items: [{ autogenerate: { directory: 'sprints' } }],
				},
			],
		}),
	],
});
