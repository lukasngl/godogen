vim.lsp.config["godogen-language-server"] = {
	-- Command and arguments to start the server.
	cmd = { "go", "run", "github.com/lukasngl/godogen-language-server" },
	-- Filetypes to automatically attach to.
	filetypes = { "go", "cucumber" },
	-- Sets the "workspace" to the directory where any of these files is found.
	-- Files that share a root directory will reuse the LSP server connection.
	-- Nested lists indicate equal priority, see |vim.lsp.Config|.
	root_markers = { "go.mod", ".git" },
	-- Specific settings to send to the server. The schema is server-defined.
	-- Example: https://raw.githubusercontent.com/LuaLS/vscode-lua/master/setting/schema.json
	settings = {},
}

vim.lsp.enable("godogen-language-server")
