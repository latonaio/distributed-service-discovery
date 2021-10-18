# distributed-service-discovery  
distributed-service-discoveryは、エッジネットワーク上のノードの死活管理を行うマイクロサービスです。

## セットアップ ##
[1] マウントされた該当ストレージ領域における任意のディレクトリで、本マイクロサービスをクローンする

[2] マイクロサービスがクローンされているディレクトリ直下で、下記コマンドを実行する

```
sudo apt-get install libpcap0.8-dev
```

[3] マイクロサービスがクローンされているディレクトリ直下で、下記コマンドを実行する

```
go build
```

[4] 下記コマンドを実行し、手順2で生成されたバイナリファイルを/usr/local/binのディレクトリに移動する。

```
sudo mv distributed-service-discovery /usr/local/bin
```

## dependencies

```
$ sudo apt-get install libpcap-dev
```
