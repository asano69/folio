#!/usr/bin/env bash
declare -A commands
commands=(
	["unzip -p library/book01.cbz folio.json"]="book01.cbzのfolio.jsonの確認"
	["zip -d library/book01.cbz folio.json"]="book01.cbzからfolio.jsonを削除"
	["rm -fr data"]="folio.dbの削除"
)

# fzf に渡す形式で出力 (Description : Command)
selected=$(for cmd in "${!commands[@]}"; do
	printf "%-30s : %s\n" "${commands[$cmd]}" "$cmd"
done | fzf --prompt="Select a command: ")

# 選択がない場合は終了
[ -z "$selected" ] && exit 0

# 選択からコマンド部分だけ抽出
cmd=$(echo "$selected" | awk -F' : ' '{print $2}')

# コマンド実行
echo "Running: $cmd"
eval "$cmd"
