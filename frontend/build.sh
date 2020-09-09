if [[ $(git diff --name-only -- .) ]]; then 
	yarn build
else
	echo "nothing to do"
fi
