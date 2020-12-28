if [[ ! -d build ]]; then
	echo "initizing frontend"
	yarn install
	yarn build
	exit 0
fi

if [[ $(git diff --name-only -- .) ]]; then 
	yarn build
else
	echo "nothing to do"
fi
