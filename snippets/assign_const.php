<?php
const NAME = "splat";
global $hello;
$hello[NAME] = true;
$ut = "Hello";
$ut .= " world";
// silence errors
@$loc = func();
// but not this time
$here = func();
