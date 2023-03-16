# 学士特定課題研究で用いたデータ、ソースコードなど

## フォルダ、ファイル

- README.md
- proof-of-work-main：ブロック追加時間のヒストグラム作成などに用いたソースコード
- pure-proof-of-stake-main
- CPU-time-on-PoW：計算時間を測るためのソースコード
- CPU-time-on-PPoS
- omnetpp-PoW：シミュレーション用のソースコード
- omnetpp-PPoS：
- Simulation-data：シミュレーション結果のデータとグラフ作成に用いたipynbファイル
- final-paper：論文、発表スライドのpdfと、論文の$\LaTeX$ファイルなど

ブロック作成モデルで１ブロックずつブロック追加時間を測ったものがproof-of-work-mainで、長時間のブロック作成のうち、どの程度CPU計算を行なっているかを計測するのが、CPU-time-on-PPoWである。この2種類のフォルダ間で、ブロック作成モデルの仕組みの違いはない。Pure Proof-of-Stakeモデルにおいても同様。

OMNeT++でのシミュレーションに使用したのが、omnetpp-PoWとPPoSである。PoWでは、実際のハッシュ計算の代わりに、ガンマ分布で近似した。
