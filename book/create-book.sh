
# https://github.com/Wandmalfarbe/pandoc-latex-template

BASE_NAME=book

cat header.md > ${BASE_NAME}.md

jq -c -r '.[].page' book.json | while read i; do
    echo $i
    sed '/^---$/,/^---$/d' $i >> ${BASE_NAME}.md
done

docker run --rm --volume "${PWD}:/data" --user $(id -u):$(id -g) pandoc/extra ${BASE_NAME}.md -o ${BASE_NAME}.pdf --template eisvogel --listings