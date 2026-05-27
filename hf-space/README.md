---
title: Argos IP Classifier
emoji: 👁
colorFrom: indigo
colorTo: purple
sdk: gradio
sdk_version: 6.15.0
app_file: app.py
pinned: false
license: other
short_description: Classificador de patenteabilidade e IPC para pesquisa UFOP
tags:
  - nlp
  - classification
  - patents
  - portuguese
  - bert
  - scikit-learn
language:
  - pt
datasets:
  - custom/ufop-nit
---

# Argos IP Classifier

Classificador de patentes desenvolvido para o **NIT-UFOP** (Núcleo de Inovação Tecnológica da Universidade Federal de Ouro Preto).

## O que faz

Dado o título e resumo de uma invenção, o modelo retorna:

1. **Patenteabilidade** — probabilidade de a invenção ter potencial de depósito no INPI
2. **Categoria IPC** — uma das 8 seções da Classificação Internacional de Patentes (A..H)
3. **Alertas Art. 10 LPI** — detecta matéria excluída de patenteabilidade

## Modelos

| Modelo | Tarefa | F1 |
|--------|--------|----|
| TF-IDF + Random Forest | Patenteabilidade binária | ~0.81 |
| TF-IDF + Random Forest | IPC (8 classes) | ~0.98 |

Treinado em 770 amostras de teses/dissertações UFOP anotadas via LLM-as-annotator  
(Honovich et al., 2022 — κ = 0.286 vs heurística Go, Landis & Koch "razoável").

## Referências

- Cohen, J. (1960). Coefficient of agreement for nominal scales.
- Landis & Koch (1977). Measurement of observer agreement.
- Honovich et al. (2022). Unnatural Instructions.
- Trajtenberg, Henderson & Jaffe (1997). University vs corporate patents.
