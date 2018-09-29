# Haste

**H**TML **A**wesomely **S**imple **T**emplating **E**ngine

Haste is a really simple templating system for static HTML files.

## Templating System

The templating system uses a really simple syntax for templating out HTML via the use of custom tag names.
Here is an example of this syntax:

```html
<!-- index.haste.html -->
<div>
  <t:button>Click Here</t:button>
</div>

<!-- button.html -->
<button class="btn btn-default">{{content}}</button>
```

In the above example we have a custom tag in `index.html` named `<t:button>`. When built haste will look for a file named `button.html` and bring it in, Replacing the custom element. The contents of the custom tag are placed into a `content` variable which can be used via double curly braces: `{{content}}`. This will be replaced with the contents in the original file. The result of the above example will look like this:

```html
<div>
  <button class="btn btn-default">Click Here</button>
</div>
```

Here are some more advanced example of what you can do with the syntax:

```html
<!-- This will look for a html file with a location of 'parts/button.html', relative to the build directory -->
<t:parts.button>Click Here</t:parts.button>

<!-- This will look up a directory, At the path '../button.html', relative to the original file -->
<t::button>Click Here</t::button>

<!-- You can nest templates as much as you want -->
<!-- Templates will always search for others relative to the root build location -->
<t:table-wrap>
  <tr>
    <td>Actions</td>
    <td><t:button>Click Here</t:button></td>
  </tr>
</t:table-wrap>

<!-- If no content injection is required, template tags can be self-closing -->
<t:button/>

<!-- Included CSS and JS files will have their contents injected into <style> or <script> tags -->
<t:styles.css/>
<t:styles.js/>

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
	<title>{{title}}</title>
</head>
<body style="{{bodyStyles}}">
	<t:parts.header-list>
		<span style="color:{{primary}}">{{var1}}</span>
	</t:parts.header-list>
</body>
</html>
```

Variables must be defined on the first lines of a file with no whitespace proceeding the starting `@` symbol. It's one variable per line in the format `@name=value`. Variables can then be used via double curly braces in the format `{{name}}`. Again, Watch any whitespace you enter in the tags as any differences to the declarations will not be forgiven.

Variables will pass down through template files but will not pass back up so parent template files will not see variables defined in child templates. Child variables inherit parent variables and will overwrite any existing if redefined.

#### Variable Injection via Attributes

Variables can be injected into child templates via the use of attributes on the template tag. For example, in the HTML below the variable named `author` will be available as a variable to the child template `book` with a value of `Dan Brown`.

```html
  <t:book author="Dan Brown"></t:book>
```

#### Variable Injection via Tags

Sometimes you may want to inject more than a simple string, Maybe a whole block of HTML. To do this you can used variable tags.
These are tags which are similar to custom template tags but follow the syntax `<v:[variable-name]>`. For example, in the HTML below the `img` and `p` HTML content will be passed to the parent template as a `person` variable.

```html
<t:my-layout>
    <v:person>
        <img src="/images/me.png">
        <p>My name is Dan</p>
    </v:person>
</t:my-layout>
```

Variable tags can only be used within custom template tags.

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
| -l   |         | Disable livereload (When watching) |
| -p   | 8081    | Port to listen on (When watching) |
| -d   | ./dist/ | Output folder for generated content |
| -r   | ./      | Relative root folder for template references |
| -v   |         | Show verbose output |


#### Usage Examples

``` bash
# Build *.haste.html files out to a ./dist/ folder
./haste

# Build *.haste.html files out to a ./dist/ folder and watch for changes
./haste -w

# Build ./src/*.haste.html files out to the ./out/ folder.
./haste -r src/ -d out/
```

## Issues and Contribution

Haste is in its early days at the moment and I'm no golang pro so bugs are highly likely, Especially while my tests are sparse. Feel free to create an issue or create a pull request.

## Testing

This code can be tested by running:

```bash
go test ./...
```

As of writing, Only the core build process is tested. Other testing is in progress.

## License and Attribution

[Haste is licensed under the MIT license.](https://raw.githubusercontent.com/ssddanbrown/haste/master/LICENSE)

This software includes works from the following great open source projects:

* [Golang](https://github.com/golang/go) - [License](https://github.com/golang/go/blob/master/LICENSE)
* [Go Color](https://github.com/fatih/color) - [License](https://github.com/fatih/color/blob/master/LICENSE.md)
* [Go fsnotify](https://github.com/howeyc/fsnotify) - [License](https://github.com/howeyc/fsnotify/blob/master/LICENSE)
* [Livereload](https://github.com/livereload/livereload-js) - [License](https://github.com/livereload/livereload-js/blob/master/LICENSE)
