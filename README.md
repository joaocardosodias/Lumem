# Inteli - Instituto de Tecnologia e Liderança

<p align="center">
<a href= "https://www.inteli.edu.br/"><img src="https://res.cloudinary.com/dwewomj84/image/upload/v1761355911/inteli_lfuayp.png" alt="Inteli - Instituto de Tecnologia e Liderança" border="0" width=40% height=40%></a>
</p>

<br>

# Manutenção Preditiva de Portas de Plataforma (PSDs)

## Subway Surfers

## Integrantes:

- <a href="https://www.linkedin.com/in/joseph-marcelo-mansur/">Joseph Mansur</a>
- <a href="https://www.linkedin.com/in/jo%C3%A3o-pedro-gon%C3%A7alves-corr%C3%AAa-araujo/">João Pedro Gonçalves Corrêa Araujo</a>
- <a href="https://www.linkedin.com/in/maria-vit%C3%B3ria-dos-santos/">Maria Vitória dos Santos</a>
- <a href="https://www.linkedin.com/in/maria-arielly/">Maria Arielly</a>
- <a href="https://www.linkedin.com/in/kaian-moura-56b8871b4/">Kaian Moura</a>
- <a href="https://www.linkedin.com/in/leandro-precaro-barankiewicz-filho/">Leandro Precaro Barankiewicz Filho</a>
- <a href="https://www.linkedin.com/in/jo%C3%A3ocardosodias/">João Cardoso Dias</a>

## Professores:

### Orientador(a)

- <a href="https://www.linkedin.com/in/murilo-zanini-de-carvalho-0980415b/">Murilo Zanini de Carvalho</a>

### Instrutores

- <a href="https://www.linkedin.com/in/diogo-martins-gon%C3%A7alves-de-morais-96404732/">Diogo Martins Gonçalves de Morais</a>
- <a href="https://www.linkedin.com/in/fabiana-iegawa/">Fabiana Iegawa</a>
- <a href="https://www.linkedin.com/in/filipe-gon%C3%A7alves-08a55015b/">Filipe Gonçalves</a>
- <a href="https://www.linkedin.com/in/jefferson-o-silva/">Jefferson Silva</a>
- <a href="https://www.linkedin.com/in/marianavioli/">Mariana Vicente Oliveira</a>
- <a href="https://www.linkedin.com/in/rafaeldonairelima/">Rafael Donaire</a>

## Descrição

&emsp; Este projeto, desenvolvido em parceria com a **Motiva** (maior empresa de infraestrutura de mobilidade do Brasil), tem como objetivo construir uma solução de **Manutenção Preditiva** para as Portas de Plataforma (*Platform Screen Doors* — PSDs) operadas nas linhas de metrô sob concessão da concessionária.<br><br>&emsp; Historicamente, a manutenção de ativos ferroviários e metroviários atua de forma preventiva periódica e corretiva emergencial quando ocorrem falhas ou alarmes em operação. O objetivo deste trabalho é realizar a virada de chave para a manutenção preditiva orientada a dados: através do tratamento de dados históricos de logs de alarmes, ordens do SAP (ZM01, ZM02 e ZM03) e movimentação de passageiros, o modelo de Inteligência Artificial identifica padrões de degradação nos componentes mecânicos e elétricos (como sensores `LSD` e roletes), antecipando falhas iminentes. A solução visa otimizar as janelas noturnas de manutenção (4 horas), aumentar a segurança operacional e reduzir paradas não programadas no transporte público.

## Documentação Adicional (Deploy do Astro / Starlight)

Clique [aqui](https://graduacao.pages.git.inteli.edu.br/2026-2a/t16/g04/) para acessar a documentação completa do projeto.

## Estrutura de pastas

```
├── assets/ --> Arquivos de imagens, diagramas e recursos visuais da documentação.
├── docs/ --> Código fonte do portal de documentação estática (Astro / Starlight).
│   ├── src/
│   │   ├── assets/
│   │   └── content/docs/
│   └── astro.config.mjs
├── eda/ --> Notebooks de Análise Exploratória de Dados, teste de hipóteses e scripts de limpeza.
│   ├── cleaning/
│   └── hipoteses/
├── src/ --> Todo o código fonte dos modelos preditivos e scripts de engenharia de recursos.
└── README.md --> Arquivo guia de apresentação do projeto (o mesmo que você está lendo agora).
```

## Instalação

&emsp; Para instalar as dependências do portal de documentação (Astro), navegue até a pasta `docs` e rode o comando:

```bash
cd docs
npm install
```

## Execução do Projeto

### 1. Documentação (Astro / Starlight)

&emsp; A documentação do projeto é gerada através do framework Astro.

&emsp; Para iniciar o servidor local de desenvolvimento e visualizar a documentação interativa:

```bash
cd docs
npm run dev
```

&emsp; Acesse a página `http://localhost:4321` no navegador.

### 2. Análise de Dados e Limpeza (Python / Jupyter Notebooks)

&emsp; As etapas de higienização de dados e testes de hipóteses estão centralizadas no diretório `eda/`.

**Passo a passo para execução dos notebooks:**

1. Crie e ative um ambiente virtual Python:

```bash
python3 -m venv venv
source venv/bin/activate
```

2. Instale as dependências para análise de dados:

```bash
pip install pandas numpy matplotlib seaborn jupyter
```

3. Abra o ambiente Jupyter Notebook para rodar os pipelines de limpeza:

```bash
jupyter notebook eda/cleaning/
```

## Histórico de lançamentos

- 0.1.0 - 14/08/2026
  - [sprint 1] Entendimento do negócio (Motiva), alinhamento com ODS, definição de personas (Ronaldo e Marta), formulação e validação estatística de 7 hipóteses de falha, pipelines de limpeza de dados (Alarmes, ZM01, ZM02, ZM03) e definição das features candidatas.
- 0.2.0 - Sprint 2 (A realizar)
  - [sprint 2] Mapeamento do fluxo de dados e desenvolvimento preliminar do modelo de machine learning/pipeline preditivo.
- 0.3.0 - Sprint 3 (A realizar)
  - [sprint 3] Treinamento e avaliação dos modelos de inteligência artificial.
- 0.4.0 - Sprint 4 (A realizar)
  - [sprint 4] Integração do modelo com a interface visual e relatórios operacionais.
- 0.5.0 - Sprint 5 (A realizar)
  - [sprint 5] Entrega final do sistema preditivo e validação de impacto operacional.

## Licença

<img style="height:22px!important;margin-left:3px;vertical-align:text-bottom;" src="https://mirrors.creativecommons.org/presskit/icons/cc.svg?ref=chooser-v1"><img style="height:22px!important;margin-left:3px;vertical-align:text-bottom;" src="https://mirrors.creativecommons.org/presskit/icons/by.svg?ref=chooser-v1"><p xmlns:cc="http://creativecommons.org/ns#" xmlns:dct="http://purl.org/dc/terms/">GRUPO 4 by [Inteli](https://www.inteli.edu.br/), [Arielly Lima](https://www.linkedin.com/in/arielly-lima/), [João Araújo](https://www.linkedin.com/in/joao-araujo/), [João Dias](https://www.linkedin.com/in/joao-dias/), [Joseph Mansur](https://www.linkedin.com/in/joseph-mansur/), [Kaian Santos Moura](https://www.linkedin.com/in/kaian-moura/), [Leandro Filho](https://www.linkedin.com/in/leandro-filho/), [Maria Santos](https://www.linkedin.com/in/maria-santos/) is licensed under <a href="http://creativecommons.org/licenses/by/4.0/?ref=chooser-v1" target="_blank" rel="license noopener noreferrer" style="display:inline-block;">Attribution 4.0 International</a>.</p>
