gen-docs:
	rm -fr docs
	cd workspace/docs && hugo
	mv workspace/docs/public docs

gen-book:
	cd book && ./create-book.sh