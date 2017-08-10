# Alertmanager

对开源项目github/prometheus/alertmanager增加了持久化存储的功能：

采用mysql作为持久化存储，将报警（Alerts）和抑制（Silences）信息存储在mysql中，确保重启后之前的数据不丢失。

官方文件链接:https://github.com/prometheus/alertmanager
