#!/bin/bash

dir="./storage"

echo "Collecting files from $dir"
total=$(ls -1 "$dir"/*.png | wc -l)
count=0

draw_progress_bar() {
    # $1 - current progress, $2 - total
    local progress=$(( ($1 * 100) / $2 ))
    local filled=$(( ($progress * 50) / 100 ))
    local empty=$(( 50 - $filled ))

    printf "\r$1 of $2 - ["
    printf "%${filled}s" | tr ' ' '#'
    printf "%${empty}s" | tr ' ' '-'
    printf "] - $progress%%"
}

echo "Processing files..."

for file in "$dir"/*.png; do
    pngquant --force --ext .png 256 "$file"

    count=$((count + 1))

    draw_progress_bar $count $total
done

echo -e "\nProcessing complete."
