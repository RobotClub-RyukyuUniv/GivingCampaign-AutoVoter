# GivingCmapaingを楽にするために自動化させるソフト

## 前提
- 今の所python環境がないとできない
- 認証以降は手動でやってもらう

## インストール
- Chromeをインストール
    - バージョンを確認
- WebDriverをインストール(Chromeにあうversion)
数字をChromeのバージョンにして
`pip install chromedriver-binary==130.0.6723.58`


## 使い方
- ドキュメント（main.pyの一番最初のところ）に定数の説明が書いているから、自分の情報に書き換える

1. 毎回URLと、団体名を書き換える
    - `URLはkumamoto-u.2024.giving-campaign.jp/form/vote/step1`のように/form/vote/step1をつける
2. main.pyを実行
3. 開かれたChromeでsms認証をして応援メッセージを書く
4. 終了したければ、ターミナルでyを入力



## 展望
GUI化して、複数のサイトを一気に開いて入力させたい
