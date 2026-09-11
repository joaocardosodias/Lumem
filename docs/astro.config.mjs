// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import mermaid from 'astro-mermaid';

const backendSidebar = {
	label: 'Backend',
	collapsed: true,
	items: [
		{ label: 'Visão Geral do Backend', slug: 'sprints/sprint-3/backend' },
		{ label: 'Práticas de Engenharia', slug: 'sprints/sprint-3/backend/praticas-de-engenharia' },
		{
			label: 'Arquitetura',
			collapsed: true,
			items: [
				{ label: 'Visão Geral da Arquitetura', slug: 'sprints/sprint-3/backend/arquitetura/visao-geral' },
				{ label: 'Clean Architecture', slug: 'sprints/sprint-3/backend/arquitetura/clean-architecture' },
				{ label: 'Banco de Dados & Schemas', slug: 'sprints/sprint-3/backend/arquitetura/banco-de-dados' },
				{ label: 'Fluxo de Ingestão', slug: 'sprints/sprint-3/backend/arquitetura/fluxo-de-ingestao' },
				{
					label: 'Decisões de Arquitetura (ADRs)',
					collapsed: true,
					items: [
						{ label: 'ADR 001 — Clean Architecture', slug: 'sprints/sprint-3/backend/arquitetura/decisoes/adr-001-clean-architecture' },
						{ label: 'ADR 002 — Fila River', slug: 'sprints/sprint-3/backend/arquitetura/decisoes/adr-002-river-postgresql' },
						{ label: 'ADR 003 — Armazenamento S3', slug: 'sprints/sprint-3/backend/arquitetura/decisoes/adr-003-object-storage' },
						{ label: 'ADR 004 — OpenTelemetry & Tracing', slug: 'sprints/sprint-3/backend/arquitetura/decisoes/adr-004-opentelemetry-tempo' },
					],
				},
			],
		},
		{
			label: 'Serviços & Qualidade',
			collapsed: true,
			items: [
				{ label: 'API REST & Contratos', slug: 'sprints/sprint-3/backend/api/openapi' },
				{ label: 'Observabilidade & Métricas', slug: 'sprints/sprint-3/backend/operacao/observabilidade' },
				{ label: 'Testes Automatizados & CI', slug: 'sprints/sprint-3/backend/qualidade/testes-e-ci' },
				{ label: 'SonarQube & Segurança', slug: 'sprints/sprint-3/backend/qualidade/sonarqube-e-seguranca' },
			],
		},
		{
			label: 'DevOps & Referências',
			collapsed: true,
			items: [
				{ label: 'DevOps & Deploy', slug: 'sprints/sprint-3/backend/operacao/configuracao-e-deploy' },
				{ label: 'Diagnóstico & Troubleshooting', slug: 'sprints/sprint-3/backend/operacao/troubleshooting' },
				{ label: 'Variáveis de Ambiente', slug: 'sprints/sprint-3/backend/referencia/variaveis-de-ambiente' },
			],
		},
	],
};

// https://astro.build/config
export default defineConfig({
	site: 'https://graduacao.pages.git.inteli.edu.br',
	base: '/2026-2a/t16/g04',
	redirects: {
		'/': '/2026-2a/t16/g04/readme',
	},
	integrations: [
		mermaid({
			theme: 'neutral',
			autoTheme: true,
		}),
		starlight({
			title: 'Documentação',
			locales: {
				root: { label: 'Português', lang: 'pt-BR' },
			},
			tableOfContents: { minHeadingLevel: 2, maxHeadingLevel: 3 },
			logo: {
				light: './src/assets/logo.png',
				dark: './src/assets/logo-white.png',
			},
			favicon: '/favicon.png',
			customCss: [
				'./src/styles/custom.css',
			],
			social: [{ icon: 'gitlab', label: 'GitLab do projeto', href: 'https://git.inteli.edu.br/graduacao/2026-2a/t16/g04' }],
			sidebar: [
				{ label: 'Solução', slug: 'readme' },

				{
					label: 'Sprints',
					collapsed: true,
					items: [
						{
							label: 'Sprint 1',
							collapsed: true,
							items: [
								{ label: 'Visão Geral da Sprint 1', slug: 'sprints/sprint-1' },
								{ label: 'Análise de Valor e da Solução', slug: 'sprints/sprint-1/analise-de-valor-e-da-solução' },
								{ label: 'Alinhamento ODS', slug: 'sprints/sprint-1/alinhamento-ods' },
								{ label: 'Personas', slug: 'sprints/sprint-1/personas' },
								{
									label: 'Hipóteses',
									collapsed: true,
									items: [{ autogenerate: { directory: 'sprints/Sprint 1/Hipoteses' } }],
								},
								{
									label: 'Limpeza de dados',
									collapsed: true,
									items: [{ autogenerate: { directory: 'sprints/Sprint 1/Limpeza de dados' } }],
								},
								{ label: 'Features Candidatas', slug: 'sprints/sprint-1/features-candidatas' },
								{ label: 'Resumo para Prova', slug: 'sprints/sprint-1/resumo-para-prova' },
							],
						},
						{
							label: 'Sprint 2',
							collapsed: true,
							items: [
								{ label: 'Visão Geral da Sprint 2', slug: 'sprints/sprint-2' },
								{ label: 'Comparação dos Modelos', slug: 'sprints/sprint-2/comparacao_modelos' },
								{ label: 'Wireframe', slug: 'sprints/sprint-2/wireframe' },
								{
									label: 'Modelos Preditivos',
									collapsed: true,
									items: [{ autogenerate: { directory: 'sprints/Sprint 2/modelos' } }],
								},
							],
						},
						{
							label: 'Sprint 3',
							collapsed: true,
							items: [
								{ label: 'Protótipo de Alta Fidelidade', slug: 'sprints/sprint-3/prototipo-alta-fidelidade' },
								{ label: 'Arquitetura UML', slug: 'sprints/sprint-3/arquitetura-uml' },
								{ label: 'Frontend', slug: 'sprints/sprint-3/frontend' },
								{ label: 'Requisitos do Sistema', slug: 'sprints/sprint-3/requisitos' },
								backendSidebar,
								{
									label: 'Modelos Preditivos',
									collapsed: true,
									items: [{ autogenerate: { directory: 'sprints/Sprint 3/modelos' } }],
								},
								{
									label: 'API de Predição',
									collapsed: true,
									items: [{ autogenerate: { directory: 'sprints/Sprint 3/api' } }],
								},
							],
						},
						{
							label: 'Sprint 4',
							collapsed: true,
							items: [{ autogenerate: { directory: 'sprints/Sprint 4' } }],
						},
						{
							label: 'Sprint 5',
							collapsed: true,
							items: [{ autogenerate: { directory: 'sprints/Sprint 5' } }],
						},
					],
				},
			],
		}),
	],
});
