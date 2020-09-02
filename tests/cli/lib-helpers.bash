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

# $1: data
# $2: expected data
compjson() {
    data="$(echo "$1" | jq -cS '.')"
    expected="$(echo "$2" | jq -cS '.')"

	if [[ "$data" != "$expected" ]]; then
		echo "Exected data:"
		echo "$data" | hexdump -C | head
		echo "Got data:"
		echo "$expected" | hexdump -C | head
		return 1
	fi
	return 0
}

