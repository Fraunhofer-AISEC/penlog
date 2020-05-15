# $1: data
# $2: expected data
compstr() {
	if [[ "$1" != "$2" ]]; then
		echo "Exected string:"
		echo "$2" | hexdump -C | head
		echo "Got string:"
		echo "$1" | hexdump -C | head
		return 1
	fi
	return 0
}