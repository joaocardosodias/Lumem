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
					items: [
						{
							label: 'Sprint 1',
							items: [
								{ label: 'Análise de Valor e da Solução', slug: 'sprints/sprint-1/analise-de-valor-e-da-solução' },
								{ label: 'Alinhamento ODS', slug: 'sprints/sprint-1/alinhamento-ods' },
								{ label: 'Hipóteses', items: [{ autogenerate: { directory: 'sprints/Sprint 1/Hipoteses' } }] },
								{ label: 'Limpeza de dados', items: [{ autogenerate: { directory: 'sprints/Sprint 1/Limpeza de dados' } }] },
							],
						},
						{ label: 'Sprint 2', items: [{ autogenerate: { directory: 'sprints/Sprint 2' } }] },
						{ label: 'Sprint 3', items: [{ autogenerate: { directory: 'sprints/Sprint 3' } }] },
						{ label: 'Sprint 4', items: [{ autogenerate: { directory: 'sprints/Sprint 4' } }] },
						{ label: 'Sprint 5', items: [{ autogenerate: { directory: 'sprints/Sprint 5' } }] },
					],
				},
			],
		}),
	],
});
