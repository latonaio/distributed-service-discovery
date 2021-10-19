# distributed-service-discovery  
distributed-service-discovery は、エッジネットワーク上のノードの死活管理(ネットワークと接続されているかどうか)を行うマイクロサービスです。  
distributed-service-discovery は、コンテナ上で稼働せず、OSレイヤーで稼働します。  
distributed-service-discovery が OSレイヤーで稼働する理由は、エッジコンピューティング環境においてコンテナオーケストレーションシステムが単一障害点とならないようにするためです。  

## 依存関係

- [gossip-propagation-d](https://github.com/latonaio/gossip-propagation-d)    
- [titaniadb-sentinel](https://github.com/latonaio/titaniadb-sentinel)  

## セットアップ ##
[1] マウントされた該当ストレージ領域における任意のディレクトリで、本マイクロサービスをクローンします

[2] マイクロサービスがクローンされているディレクトリ直下で、下記コマンドを実行します

```
sudo apt-get install libpcap0.8-dev
```

[3] マイクロサービスがクローンされているディレクトリ直下で、下記コマンドを実行します

```
go build
```

[4] 下記コマンドを実行し、手順2で生成されたバイナリファイルを/usr/local/binのディレクトリに移動します

```
sudo mv distributed-service-discovery /usr/local/bin
```

