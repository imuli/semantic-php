<?php

const NAME = "splat";

require_once('header.php');
require('functions.php');

// might as well set our namespace
namespace Something;
use Nothing;

// set a global
global $hello;

$hello[NAME] = true;

// if we're ...
if($hello){
	echo "Hello World";
} else {
	echo "Goodbye";
}

function func() {
	return [1, 2, 3];
}

// silence errors
@$loc = func();
// but not this time
$here = func();

foreach($loc as $l){
	// do something with $l
}

for($i = 0; $i < 10; $i++){
	echo $i;
}

$here->where();

print_footer();

