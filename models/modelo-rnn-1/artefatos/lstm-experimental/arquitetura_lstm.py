"""Arquitetura dos pesos salvos em pesos_lstm.pt."""
import torch
from torch import nn


class LSTMPortas(nn.Module):
    def __init__(self, n_temporais, n_estaticas):
        super().__init__()
        self.lstm = nn.LSTM(input_size=n_temporais, hidden_size=64, batch_first=True)
        self.normalizacao = nn.LayerNorm(64)
        self.dropout_temporal = nn.Dropout(0.3)
        self.dense = nn.Linear(64 + n_estaticas, 32)
        self.relu = nn.ReLU()
        self.dropout_dense = nn.Dropout(0.3)
        self.saida = nn.Linear(32, 1)
        self.sigmoid = nn.Sigmoid()

    def forward(self, temporal, estatico):
        historico, _ = self.lstm(temporal)
        ultimo_dia = historico[:, -1, :]
        resumo = self.dropout_temporal(self.normalizacao(ultimo_dia))
        combinado = torch.cat([resumo, estatico], dim=1)
        combinado = self.dropout_dense(self.relu(self.dense(combinado)))
        return self.sigmoid(self.saida(combinado))
