# Haste

**H**TML **A**wesomely **S**imple **T**emplating **E**ngine

Haste is a really simple templating system for static HTML files.

## Templating System

The templating system uses a really simple syntax for templating out HTML via the use of custom tag names.
Here is an example of this syntax:

```html
<!-- index.html -->
<div>
  <t:button>Click Here</t:button>
</div>

<!-- button.html -->
<button class="btn btn-default">@content</button>
```

In the above example we have a custom tag in `index.html` named `<t:button>`. When built haste will look for a file named `button.html` and bring it in, Replacing the custom element. The contents of the custom tag are marked with the text `@content`. This will be replaced with the contents in the original file. The result of the above example will look like this:

```html
<div>
  <button class="btn btn-default">Click Here</button>
</div>
```

Here are some more advanced example of what you can do with the syntax:

```html
<!-- This will look for a html file with a location of 'parts/button.html', relative to the original file -->
<t:parts.button>Click Here</t:parts.button>

<!-- This will look up a directory, At the path '../button.html', relative to the orginal file -->
<t::button>Click Here</t::button>

<!-- You can nest templates as much as you want -->
<!-- Templates will always search for others relative to their own file location -->
<t:table-wrap>
  <tr>
    <td>Actions</td>
    <td><t:button>Click Here</t:button></td>
  </tr>
</t:table-wrap>

<!-- If no content injection is required, template tags can be self-closing -->
<t:button/>

```

### Variables

You can have simple name, value pairs of variables in your templates. These are defined and used in the following format:

```html
@var1=This is my first variable
@title=Haste Templating System
@bodyStyles=padding:0;margin:0;font-size:16px;
@primary=#00ACED
<html lang="en">
<head>
	<title>{{{title}}}</title>
</head>
<body style="{{{bodyStyles}}}">
	<t:parts.header-list>
		<span style="color:{{{primary}}}">{{{var1}}}</span>
	</t:parts.header-list>
</body>
</html>
```

Variables must be defined on the first lines of a file with no whitespace proceeding the starting `@` symbol. It's one variable per line in the format `@name=value`. Variables can then be used via triple braces in the format `{{{name}}}`. Again, Watch any whitespace you enter in the tags as any differences to the declarations will not be forgiven.

Variables will pass down through template files but will not pass back up so parent template files will not see variables defined in child templates. Child variables inherit parent variables and will overwrite any existing if redefined.

## Command Line Usage

Download the relevant executable file for your platform from the [latest release page](https://github.com/ssddanbrown/haste/releases/latest) and ensure it has executable permissions. Rename the executable to haste to make it quicker to run. Place haste either local to your HTML file or move it somewhere in your path so it can be executed globally.

By default the application outputs to the command line. With the `-w` flag files can be watched and auto-built when changed. Additionally you can enable livereload with the `-l` flag which will auto-reload the browser on change.

```bash
./haste [OPTIONS] file
```

#### Available Options

| Flag | Default | Description |
|------|---------|-------------|
| -w   |         | Watch file for changes and auto-compile on change. <br> Outputs to `<filename>.gen.html`.  <br> Starts a http server for file serving and opens the browser automatically.    |
| -l   |         | Enable livereload (When watching) |
| -p   | 8081    | Port to listen on (When watching) |
| -d   | 2       | Folder depth to watch for changes (When watching) |
| -v   |         | Show verbose output |


#### Usage Examples

``` bash
# Build index.html and output the result to the command line.
./haste index.html

# Build index.html save output to index.build.html
./haste index.html > index.build.html

# Watch index.html and build on change whilst also enabling livereload
./haste -w -l index.html

# As above but listen on port 80 (Instead of the 8081 default)
# and listen to file changes up to 5 folder levels deep.
./haste -w -l -p 80 -d 5 index.html

```

## Issues and Contribution

Haste is in its early days at the moment and I'm no golang pro so bugs are highly likely, Especially while my tests are sparse. Feel free to create an issue or create a pull request.

## License and Attribution

[Haste is licensed under the MIT license.](https://raw.githubusercontent.com/ssddanbrown/haste/master/LICENSE)

This software includes works from the following great open source projects:

* [Golang](https://github.com/golang/go) - [License](https://github.com/golang/go/blob/master/LICENSE)
* [Go Rice](https://github.com/GeertJohan/go.rice) - [License](https://github.com/GeertJohan/go.rice/blob/master/LICENSE)
* [Livereload](https://github.com/livereload/livereload-js) - [License](https://github.com/livereload/livereload-js/blob/master/LICENSE)
