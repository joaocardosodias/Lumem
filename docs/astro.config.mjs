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

const fallbackSite = 'https://graduacao.pages.git.inteli.edu.br';
const fallbackBase = '/2026-2a/t16/g04';
const pagesUrl = process.env.CI_PAGES_URL;
const site = pagesUrl ? new URL(pagesUrl).origin : fallbackSite;
const base = pagesUrl
	? (new URL(pagesUrl).pathname.replace(/\/$/, '') || '/')
	: fallbackBase;
const readmePath = `${base === '/' ? '' : base}/readme`;
const basePrefix = base === '/' ? '' : base.replace(/\/$/, '');

/** Prefix site-root links (`/sprints/...`) with Astro `base` so GitLab Pages works. */
function rehypePrefixBase() {
	return (tree) => {
		walk(tree);
		function walk(node) {
			if (node.type === 'element' && node.tagName === 'a' && typeof node.properties?.href === 'string') {
				const href = node.properties.href;
				if (
					basePrefix &&
					href.startsWith('/') &&
					!href.startsWith('//') &&
					href !== basePrefix &&
					!href.startsWith(`${basePrefix}/`)
				) {
					node.properties.href = `${basePrefix}${href}`;
				}
			}
			if (Array.isArray(node.children)) node.children.forEach(walk);
		}
	};
}

function rehypeNumberedTitles() {
	return (tree, file) => {
		if (!file.path?.replaceAll('\\', '/').includes('/content/docs/sprints/')) return;
		let section = 0;
		const text = (node) => node.value ?? (node.children ?? []).map(text).join('');
		function walk(node) {
			const heading = node.type === 'element' && node.tagName === 'h2';
			const existingBanner = node.type === 'mdxJsxFlowElement' && node.name === 'h2';
			if (heading || existingBanner) {
				const title = text(node).trim();
				const number = /^(\d+)(?:\.\d+)*(?:[.)]\s+|\s+)/.exec(title);
				const subtopic = /^\d+\.\d+/.test(title);
				section = number ? Number(number[1]) : section + 1;
				if (heading && !subtopic) {
					// Keep generated IDs intact so existing links and the TOC still work.
					if (!number) node.children.unshift({ type: 'text', value: `${section}. ` });
					node.properties ??= {};
					const classes = node.properties.className ?? [];
					node.properties.className = [...(Array.isArray(classes) ? classes : [classes]), 'group-numbered-title'];
				}
			}
			if (Array.isArray(node.children)) node.children.forEach(walk);
		}
		walk(tree);
	};
}

// https://astro.build/config
export default defineConfig({
	site,
	base,
	redirects: {
		'/': readmePath,
		'/sprints/sprint-5/mapeamento-testes-rf': `${basePrefix}/sprints/sprint-5/implementacao-validacao-testes/#3-mapeamento-e-situação-atual-dos-requisitos`,
		'/sprints/sprint-5/validacao-autenticacao-acesso': `${basePrefix}/sprints/sprint-5/implementacao-validacao-testes/#4-autenticação-e-acesso--rf01-e-rf02`,
		'/sprints/sprint-5/validacao-importacao-dados': `${basePrefix}/sprints/sprint-5/implementacao-validacao-testes/#5-importação-de-dados--rf03-e-rf04`,
		'/sprints/sprint-5/validacao-cadastro-consulta-filtros': `${basePrefix}/sprints/sprint-5/implementacao-validacao-testes/#6-cadastro-consulta-e-filtros--rf05-e-rf06`,
		'/sprints/sprint-5/validacao-execucao-horizonte-predicao': `${basePrefix}/sprints/sprint-5/implementacao-validacao-testes/#7-execução-e-horizonte-da-predição--rf07-e-rf08`,
		'/sprints/sprint-5/validacao-classificacao-portas': `${basePrefix}/sprints/sprint-5/implementacao-validacao-testes/#8-classificação-das-portas--rf09`,
		'/sprints/sprint-5/validacao-dashboard-inspecoes': `${basePrefix}/sprints/sprint-5/implementacao-validacao-testes/#9-dashboard-e-inspeções--rf10-e-rf11`,
		'/sprints/sprint-5/validacao-integracoes-fluxo-completo': `${basePrefix}/sprints/sprint-5/implementacao-validacao-testes/#10-integrações-e-fluxo-operacional`,
		'/sprints/sprint-5/validacao-postgresql-regressao-completa': `${basePrefix}/sprints/sprint-5/implementacao-validacao-testes/#12-postgresql-real-e-regressão-completa`,
	},
	markdown: {
		rehypePlugins: [rehypePrefixBase, rehypeNumberedTitles],
	},
	integrations: [
		mermaid({
			theme: 'neutral',
			autoTheme: true,
		}),
		starlight({
			components: { Head: './src/components/DocumentationHead.astro' },
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
								{
									label: 'Modelos Preditivos',
									collapsed: true,
									items: [
										{ autogenerate: { directory: 'sprints/Sprint 3/modelos' } },
									],
								},
								{ label: 'Arquitetura UML', slug: 'sprints/sprint-3/arquitetura-uml' },
								{ label: 'Requisitos do Sistema', slug: 'sprints/sprint-3/requisitos' },
								{ label: 'Protótipo de Alta Fidelidade', slug: 'sprints/sprint-3/prototipo-alta-fidelidade' },
								{ label: 'Frontend', slug: 'sprints/sprint-3/frontend' },
								backendSidebar,
								{
									label: 'API de Predição',
									collapsed: true,
									items: [{ autogenerate: { directory: 'sprints/Sprint 3/api' } }],
								},
								{ label: 'Análise Financeira', slug: 'sprints/sprint-3/analise_financeira_projeto_poc_implementacao' },
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
