#!/bin/sh
# gcvote の最新版をインストールする（macOS / Linux）。
#   curl -fsSL https://robotclub-ryukyuuniv.github.io/GivingCampaign-AutoVoter/install.sh | sh
# インストール先は環境変数 GCVOTE_INSTALL_DIR で変更できる（既定: ~/.local/bin）。
set -eu

repo="RobotClub-RyukyuUniv/GivingCampaign-AutoVoter"
dir="${GCVOTE_INSTALL_DIR:-$HOME/.local/bin}"

case "$(uname -s)" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *) echo "未対応のOSです: $(uname -s)（Windows は zip を手動でダウンロードしてください）" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) echo "未対応のCPUです: $(uname -m)" >&2; exit 1 ;;
esac

name="gcvote_${os}_${arch}.tar.gz"
base="https://github.com/${repo}/releases/latest/download"
tmp="$(mktemp -d)"
trap 'rm -r "$tmp"' EXIT

echo "ダウンロード中: ${name}"
curl -fsSL -o "$tmp/$name" "$base/$name"
curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt"

want="$(grep " ${name}\$" "$tmp/checksums.txt" | cut -d' ' -f1)"
if command -v sha256sum >/dev/null 2>&1; then
  got="$(sha256sum "$tmp/$name" | cut -d' ' -f1)"
else
  got="$(shasum -a 256 "$tmp/$name" | cut -d' ' -f1)"
fi
if [ -z "$want" ] || [ "$want" != "$got" ]; then
  echo "チェックサムが一致しません。ダウンロードをやり直してください。" >&2
  exit 1
fi

tar -xzf "$tmp/$name" -C "$tmp" gcvote
mkdir -p "$dir"
mv "$tmp/gcvote" "$dir/gcvote"
chmod +x "$dir/gcvote"

echo "インストールしました: $dir/gcvote ($("$dir/gcvote" version))"
case ":$PATH:" in
  *":$dir:"*) ;;
  *)
    echo ""
    echo "⚠ $dir が PATH に入っていません。シェルの設定ファイル（~/.zshrc など）に次を追加してください:"
    echo "  export PATH=\"$dir:\$PATH\""
    ;;
esac
