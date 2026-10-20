# GivingCampaign-AutoVoter (gcvote)

Giving Campaign 2026 で他大学の団体を応援（投票）する CLI です。
ブラウザを自動操作せずに、サイトが内部で使っている API を直接呼びます。
相互投票の告知文を貼り付けるだけで、書かれている団体すべてに投票します。

- Cross Campus にログインして使います。ログインしていると、氏名・連絡先などはプロフィールから自動で入ります。電話番号が認証済みなら SMS 認証も不要です。
- フォームのうちプロフィールから埋まらない項目は、初回に対話で尋ねて保存します。
  - 名前等の公開への同意
  - 卒業/修了見込み年度
  - 投票先団体との関係性
  - 知ったきっかけ
- 応援メッセージは必須です。保存したメッセージ（例：「琉球大学 Robotサークルです。応援しています！」）が毎回の既定値になり、Enter だけで送れます。
  - 告知文に「コメント欄に必ず「〜」と記入」があり、保存したメッセージにその文言が含まれていない場合は、指定に合わせた文面を既定値にします。
- 投票済み・投票受付の終了・URL の間違い（途中で切れている・去年のもの・大学や団体が存在しない）は、1件ずつ理由を表示します。

> [!NOTE]
> 公式に公開された API ではありません。サイトの更新で動かなくなることがあります。
> 応援は1つの大学につき1人1回までです。このツールは自分のアカウントでの投票を楽にするためのものです。

## インストール

**ドキュメントとダウンロード: https://robotclub-ryukyuuniv.github.io/GivingCampaign-AutoVoter/**

```sh
# macOS / Linux
curl -fsSL https://robotclub-ryukyuuniv.github.io/GivingCampaign-AutoVoter/install.sh | sh
```

```powershell
# Windows (PowerShell)
irm https://robotclub-ryukyuuniv.github.io/GivingCampaign-AutoVoter/install.ps1 | iex
```

```sh
# Go がある場合
go install github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/cmd/gcvote@latest
```

ビルド済みのファイルは [Releases](https://github.com/RobotClub-RyukyuUniv/GivingCampaign-AutoVoter/releases/latest) からもダウンロードできます。

## 使い方

```sh
gcvote
```

1. 告知文（または投票ページの URL）をそのまま貼り付けます。改行・空行はいくつあっても大丈夫です。最後に新しい行で `/end` と入力して Enter を押すと確定します。
2. 初回はブラウザが開くので、Cross Campus にログインします。セッションは保存され、次回からは不要です。
3. 初回だけ、フォームの残りの項目を尋ねられます。保存すると次回からは聞かれません。
4. 送信内容が表示されるので、規約・プライバシーポリシーに同意して送信します。

すでに投票した大学は自動で判定してスキップします。同じ告知文を何度実行しても二重投票にはなりません。全部投票済みなら、何も聞かずに終了します。

```
投票先:
  1. 滋賀県立大学 / ロボット部コンピュータ研究会（現在 199票）
  2. 東北大学 / 学友会人力飛行部Windnauts（現在 …票）
  ...
✓ 滋賀県立大学 / ロボット部コンピュータ研究会: 投票しました（メッセージ送信済み）
- 東北大学 / 学友会人力飛行部Windnauts: この大学では投票済みのためスキップ
✗ ⑤… — 団体IDが不完全です（URLが途中で切れていないか確認してください）

結果: 投票 1件 / 投票済み 1件 / エラー 1件
```

その他の例:

```sh
gcvote announce.txt                      # 告知文をファイルで渡す
gcvote 'https://usp.2026.giving-campaign.jp/groups?group=...'
gcvote -dry-run announce.txt             # 送信せず内容と投票済みかどうかだけ確認
gcvote -y announce.txt                   # 確認なしで送信（設定保存済みなら対話ゼロ）
gcvote -m '応援しています！' announce.txt   # メッセージを今回だけ変える
gcvote config                            # 保存した回答を設定し直す
gcvote login / gcvote logout             # ログインし直す / セッション削除
```

オプションは URL やファイルより前に書いてください（`gcvote -h` で一覧）。

## 設定とデータの保存場所

| 内容 | macOS | Linux |
|---|---|---|
| 回答の設定 | `~/Library/Application Support/gcvote/config.yaml` | `~/.config/gcvote/config.yaml` |
| ログインセッション | `~/Library/Application Support/gcvote/session.json` | `~/.config/gcvote/session.json` |

- どちらも本人だけが読めるパーミッション（0600）で保存します。
- 書式は [config.example.yaml](config.example.yaml) を参照してください。
- 個人情報は Cross Campus から取得するので、設定ファイルには書きません。
- リポジトリ内に `config.yaml` を置かないでください（`.gitignore` 済み）。

## ログインがうまくいかないとき

gcvote は手元に `http://127.0.0.1:<port>` を立て、ログイン後にそこへ戻ってきてもらう方式です。
戻ってこない場合（エラーページになる場合など）は、公式サイトでログインし、セッション ID を貼り付けます。

1. 投票先大学のサイトの投票フォームを開き、「Cross Campusにログインしたことがある方」の「ログイン」からログインします。
2. フォームのタブで開発者ツールのコンソールを開き、次を実行します。
   ```js
   sessionStorage.getItem("cc_session_id")
   ```
3. 表示された値を、gcvote の「セッションIDを貼り付けてEnter」に貼り付けます。

## 仕組み

すべて `https://api.alumnote.jp/gc2026` の API です。

1. `GET /gc2026_organizations/by_slug/{大学}` と `GET …/gc2026_groups` で、大学と団体を特定します。
2. `GET /auth/login` で Cross Campus にログインし、`POST /auth/exchange_token` でセッションを得ます。以降は `X-CC-Session-ID` ヘッダで送ります。
3. `GET /cross-campus/profile` でプロフィールと `auth_token` を取得します。電話番号が未認証の場合は SMS 認証を行います。
4. `POST …/gc2026_participants/other_university_bachelor_student` で参加登録します。
5. `POST …/gc2026_participants/{id}/gc2026_votes` で投票します。
6. `POST …/gc2026_votes/{id}/gc2026_message` で応援メッセージを送ります。

## 開発

```sh
go test ./...
```

リリースするには `v1.2.3` のようなタグを push します。GitHub Actions が GoReleaser で各 OS 向けにビルドし、Releases に公開します。`docs/` を変更して main に push すると、GitHub Pages が更新されます。

テストは偽の API サーバーで動くので、本番には送信しません。
`-v` を付けると通信内容が表示されます（個人情報が含まれるので共有しないこと）。

## 制限

- 対応しているのは「他大学の学生」としての投票だけです。自分の大学への投票、卒業生・教職員などの区分には未対応です。
- 寄付（もっと応援する）には対応していません。
