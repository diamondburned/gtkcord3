Yellow="\e[33m"
LightGreen="\e[92m"

if [ $# -eq 0 ]; then
	echo -e "Usage : ./builder.sh build|install"
	echo -e "build - builds the plugin in the working directory"
	echo -e "install - builds the plugin and moves it into GTKCord's plugin folder"

elif [ $1 == "build" ]; then
	echo -e "${Yellow}⏳ Building Plugin..."
	go build -buildmode=plugin

	if [[ $? == 0 ]]; then
		echo -e "${LightGreen}✅	Built Successfully"
	else
		echo -e "✖️	Failed To Build Plugin"
		exit 1
	fi

elif [ $1 == "install" ]; then
	echo -e "${Yellow}Installing..."
	go build -o ~/.config/gtkcord/plugins -buildmode=plugin

	if [[ $? == 0 ]]; then
		echo -e "${LightGreen}✅	Built Successfully"
	else
		echo -e "✖️	Failed To Build Plugin"
		exit 1
	fi
fi
