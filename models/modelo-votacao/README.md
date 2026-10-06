# Junção de modelos: votação por maioria e voto suave

Este documento explica como três modelos de previsão de falhas foram combinados em uma única resposta, por que essa combinação foi feita e o que os resultados mostram. Todo o código está no notebook `modelo_votacao.ipynb`, que fica nesta mesma pasta.

## A pergunta que os modelos respondem

Cada linha dos dados representa uma porta de plataforma em um dia específico. Para cada uma dessas linhas, os modelos tentam prever se aquela porta vai receber uma ordem de manutenção corretiva emergencial, chamada ZM01, nos sete dias seguintes. A resposta é verdadeira quando a falha acontece dentro dessa janela e falsa quando ela não acontece.

Essa pergunta é difícil porque as falhas são raras. No período de teste, que vai de 1º de maio a 31 de outubro de 2025 e reúne 71.184 linhas, apenas 8,7% das linhas terminam em falha. Por causa disso, a acurácia engana bastante. Um modelo que nunca emitisse alerta acertaria mais de 91% das linhas e mesmo assim não avisaria a manutenção sobre nenhuma falha. Por esse motivo, ao longo de todo o trabalho, as métricas que importam são a precisão, que diz quantos alertas estavam certos, o recall, que diz quantas falhas foram avisadas, e o F1, que equilibra as duas.

## Por que juntar modelos

Nenhum modelo acerta tudo, e cada um erra de um jeito. Quando os erros de modelos diferentes caem em lugares diferentes, uma combinação deles tende a corrigir parte desses erros, porque o acerto de dois compensa o engano de um. Essa ideia só funciona quando os modelos são de fato diferentes entre si. Por isso a escolha dos votantes priorizou famílias distintas de algoritmos, mesmo quando havia um candidato com desempenho individual um pouco maior.

## Os três votantes

O primeiro votante é o Random Forest do Modelo 5, o mesmo que está em produção na API. Ele é um conjunto de árvores de decisão treinadas sobre indicadores agregados de cada porta, como contagem de alarmes recentes e tempo desde a última corretiva. O voto dele usa o limiar salvo no próprio artefato de produção, que é 0,48.

O segundo votante é a RNN otimizada com informações extras. É uma rede neural recorrente do tipo GRU com duas camadas, que lê a sequência dos últimos trinta dias de cada porta e recebe também o histórico de manutenções. O limiar dela, 0,66, foi escolhido no período de validação durante a otimização.

O terceiro votante é o Modelo 8, que combina um Isolation Forest, responsável por medir o quanto o comportamento de uma porta foge do normal, com classificadores supervisionados que usam essa medida como mais uma informação. Como esse modelo não tinha um artefato salvo, o notebook o retreina com o mesmo código do notebook original. O limiar dele, 0,51, também vem da validação.

Os Modelos 6 e 7 ficaram de fora porque respondem a outra pergunta. Eles olham para o sistema de portas inteiro de uma estação, enquanto os três votantes olham para cada porta individualmente. Somar respostas a perguntas diferentes não faria sentido. O Modelo 5.1, que é o Random Forest sem a informação do número da porta, também ficou de fora, já que ele votaria quase sempre igual ao Random Forest original e não traria diversidade à votação.

## Primeira forma de juntar: votação por maioria

A primeira estratégia é a mais simples possível. Cada modelo olha para uma linha e vota sim ou não usando o próprio limiar. Quando dois ou três modelos votam sim, a votação emite um alerta. Nenhum limiar é ajustado com os dados de teste, porque cada modelo já trouxe o seu da validação, e isso garante que os resultados medem o desempenho real em dados nunca vistos.

A votação por maioria emitiu 23.482 alertas no teste, com precisão de 15,5%, recall de 58,8% e F1 de 0,246. O Random Forest sozinho, que era o melhor votante individual, chegou a precisão de 15,3%, recall de 59,2% e F1 de 0,243. A maioria ficou ligeiramente à frente, com 537 alertas a menos, e deixou de avisar 21 falhas que o Random Forest teria avisado.

O ganho foi pequeno, e a razão aparece quando se mede o quanto os votantes concordam. Random Forest e Modelo 8 votam igual em 82% das linhas, Random Forest e RNN em 81,8%, e RNN e Modelo 8 em 74,1%. Como a maioria só muda alguma coisa quando os modelos discordam, ela decide de verdade em apenas cerca de 31% das linhas. Random Forest e Modelo 8 são ambos baseados em árvores e treinados sobre o mesmo painel de dados, então tendem a errar nos mesmos lugares. A RNN é o votante mais diferente dos três.

Mesmo com esse ganho modesto, a votação revelou uma informação valiosa. Nas linhas em que nenhum modelo votou sim, só 4,2% terminaram em falha. Nas linhas em que os três votaram sim, essa taxa subiu para 18,3%. Ou seja, a quantidade de votos carrega informação sobre o risco, e isso motivou a segunda estratégia.

## Segunda forma de juntar: voto suave

A votação por maioria descarta a confiança de cada modelo. Uma porta em que o Random Forest calculou probabilidade de 0,49 recebe o mesmo voto que uma porta com probabilidade de 0,95, embora o modelo esteja muito mais seguro no segundo caso. O voto suave aproveita essa confiança.

O funcionamento é o seguinte. Em cada dia, as portas são ordenadas pela probabilidade calculada por cada modelo, e cada porta recebe a sua posição nesse ordenamento, expressa como um percentual entre zero e um. Uma porta com posição 0,95 no Random Forest, por exemplo, está entre as 5% mais arriscadas daquele dia segundo esse modelo. O score suave de cada porta é a média das três posições.

Duas razões levaram a usar a posição dentro do dia em vez da probabilidade diretamente. A primeira é que as probabilidades dos três modelos estão em escalas diferentes, como mostram os limiares de 0,48, 0,51 e 0,66, e tirar a média delas daria peso indevido ao modelo que naturalmente produz números maiores. A segunda é que a equipe de manutenção decide dia a dia quais portas inspecionar, então comparar as portas dentro do mesmo dia reflete a decisão real.

Para que a comparação fosse justa e nada fosse ajustado olhando o teste, o voto suave recebeu pesos iguais para os três modelos e a mesma quantidade de alertas por dia que a regra com a qual está sendo comparado. Se a maioria emitiu 130 alertas em um certo dia, o voto suave também emite 130 naquele dia, escolhendo as 130 portas de maior score suave. Assim a única coisa que muda é quais portas recebem o alerta.

## O que o voto suave mostrou

Como alerta de sim ou não, o voto suave ficou muito próximo da maioria e um pouco abaixo dela. Com os mesmos 23.482 alertas, ele chegou a precisão de 15,3%, recall de 57,8% e F1 de 0,241, contra 0,246 da maioria. Para esse uso específico, a maioria continua sendo a melhor regra.

A vantagem do voto suave aparece na capacidade de ordenar as portas por risco. Uma forma de medir isso, sem depender de nenhum limiar, é a área sob a curva de precisão e recall, chamada de PR AUC, que resume o quanto as portas que realmente falham ficam no topo da lista. O voto suave atingiu 0,197, acima do Random Forest (0,184), da RNN (0,181) e do Modelo 8 (0,174). A simples contagem de votos ficou em 0,146, porque ela só tem quatro valores possíveis e por isso ordena as portas de forma grosseira.

Essa melhor ordenação se traduz em ganho real quando a manutenção precisa ser mais seletiva. A unanimidade, em que os três modelos votam sim, emitiu 13.421 alertas com precisão de 18,3%, recall de 39,5% e F1 de 0,250. Com exatamente a mesma quantidade de alertas por dia, o voto suave chegou a precisão de 18,6%, recall de 40,2% e F1 de 0,254, encontrando 41 falhas a mais sem nenhum alerta extra.

O score suave também permite separar as portas em faixas de risco dentro de cada dia. Na metade de menor score, 4,0% das portas falharam nos sete dias seguintes. Entre 50% e 75%, a taxa foi de 9,9%. Entre 75% e 90%, foi de 12,5%. Nos 10% de maior score, 23,3% das portas falharam, quase seis vezes a taxa da metade de menor risco.

## Como usar esses resultados

A recomendação que sai deste trabalho combina as duas estratégias. A votação por maioria serve como alerta, porque teve o melhor F1 entre as regras de sim ou não. O voto suave serve para ordenar a fila de inspeção, porque coloca no topo as portas com maior chance real de falha. Na prática, a equipe pode começar o dia pelas portas da faixa dos 10% de maior risco, onde quase uma em cada quatro vai precisar de uma corretiva na semana.

## Limitações e próximos passos

O voto suave usa pesos iguais para os três modelos. É provável que pesos diferentes funcionem melhor, já que o Random Forest e a RNN ordenam as portas um pouco melhor que o Modelo 8. Aprender esses pesos exige previsões honestas no período de validação, que vai de 31 de janeiro a 23 de abril de 2025. Hoje isso não é possível porque o Random Forest de produção e a RNN foram treinados com dados até 23 de abril e já viram esse período. O próximo passo é retreinar os dois apenas até 23 de janeiro, gerar as previsões da validação e então ajustar os pesos e o limiar ali, aplicando o resultado ao teste só no final.

Há também duas observações sobre os números de referência. O Random Forest usado aqui é o artefato de produção, com o limiar salvo nele, e por isso seus números diferem um pouco dos que aparecem no notebook do Modelo 5. Da mesma forma, o código atual do Modelo 8 escolhe o classificador final pelo melhor F1 na validação, e o resultado documentado originalmente para esse modelo vem de uma versão anterior, com outro critério de limiar.

## Como reproduzir

Basta executar o notebook `modelo_votacao.ipynb` do início ao fim com o ambiente virtual do projeto. As previsões do Random Forest e da RNN são lidas de um arquivo já gerado pelo script de avaliação da pasta da RNN otimizada, então o notebook não precisa do PyTorch. O Modelo 8 é retreinado dentro do próprio notebook, o que leva alguns minutos.
