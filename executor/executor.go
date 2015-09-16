package executor

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/blacklabeldata/kappa/common"
	"github.com/blacklabeldata/kappa/datamodel"
	"github.com/blacklabeldata/kappa/skl"
)

func NewSession(ns string, user datamodel.User) Session {
	return Session{ns, user}
}

func NewExecutor(session Session, term common.Terminal, sys datamodel.System) *Executor {
	return &Executor{session, term, sys}
}

// Session provides session and connection related information
type Session struct {
	namespace string
	user      datamodel.User
}

// Executor executes successfully parsed queries
type Executor struct {
	session  Session
	terminal common.Terminal
	system   datamodel.System
}

// Execute processes each statement
func (e *Executor) Execute(w *common.ResponseWriter, stmt skl.Statement) {

	// Verify session has a user
	if e.session.user == nil {
		w.Fail(common.InternalServerError, "could not determine session user")
		return
	}

	switch stmt.NodeType() {
	case skl.UseNamespaceType:
		e.handleUseStatement(w, stmt)
	case skl.CreateNamespaceType:
		e.handleCreateNamespace(w, stmt)
	case skl.ShowNamespaceType:
		e.handleShowNamespace(w, stmt)
	}
}

func (e *Executor) handleUseStatement(w *common.ResponseWriter, stmt skl.Statement) {
	use, ok := stmt.(*skl.UseStatement)
	if !ok {
		w.Fail(common.InvalidStatementType, "expected *UseStatement, got %s instead", reflect.TypeOf(stmt))
		return
	}

	// Get user from session
	user := e.session.user

	// Get namespace store
	namespaceStore, err := e.system.Namespaces()
	if err != nil {
		w.Fail(common.InternalServerError, "could not access namespace data")
		return
	}

	// Get namespace
	name := use.Namespace()

	// Verify namespace existence
	_, err = namespaceStore.Get(name)
	if err == datamodel.ErrNamespaceDoesNotExist {
		w.Fail(common.NamespaceDoesNotExist, name)
		return
	} else if err != nil {
		w.Fail(common.InternalServerError, "could not access namespace data")
		return
	}

	// If the user is an admin, grant access
	if user.IsAdmin() {
		e.session.namespace = name
		e.terminal.SetPrompt(fmt.Sprintf("kappa: %s> ", name))
		w.Success(common.OK, "")
		return
	}

	// Verify user has access to the namespace or is admin
	// 		If user has access, update session namespace and terminal
	// 		Otherwise, return access denied error
	for _, namespace := range user.Namespaces() {
		if namespace == name {
			e.session.namespace = name
			e.terminal.SetPrompt(fmt.Sprintf("kappa: %s> ", name))
			w.Success(common.OK, "")
			return
		}
	}

	// Otherwise, the user is not authorized
	w.Fail(common.Unauthorized, "")
}

// Only the admin can create root namespaces.
// Admin can also create sub-namespaces for any existing namespace.
// If the user is not the admin, they must have the 'create.namespace'
//  permission for the parent namespace.
// Root namespaces don't have any periods.
func (e *Executor) handleCreateNamespace(w *common.ResponseWriter, stmt skl.Statement) {

	createStatement, ok := stmt.(*skl.CreateNamespaceStatement)
	if !ok {
		w.Fail(common.InvalidStatementType, "expected *CreateNamespaceStatement, got %s instead", reflect.TypeOf(stmt))
		return
	}

	// Get namespace store
	namespaceStore, err := e.system.Namespaces()
	if err != nil {
		w.Fail(common.InternalServerError, "could not access namespace data")
		return
	}

	// Get session user
	user := e.session.user

	// Get namespace
	namespace := createStatement.Namespace()

	// If err == nil, the namespace already existed
	if e.namespaceAlreadyExists(namespace, namespaceStore) {
		w.Success(common.NamespaceAlreadyExists, namespace)
		return
	}

	// If root namespace
	if createStatement.IsRootNamespace() {
		e.handleCreateRootNamespace(w, createStatement, namespaceStore)
		return
	}

	// Admin user is granted access by default
	access := user.IsAdmin()

	// Get parent namespace
	var parent datamodel.Namespace
	index := strings.LastIndex(namespace, ".")
	parentNamespace := namespace[:index]

	// If the user is not an admin check their permissions for the parent namespace
	if !access {

		// Get user roles for parent namespace
		roles := user.Roles(parentNamespace)

		// Determine if parent namespace exists
		ns, err := namespaceStore.Get(parentNamespace)
		if err == datamodel.ErrNamespaceDoesNotExist {
			w.Fail(common.NamespaceDoesNotExist, parentNamespace)
			return
		} else if err != nil {
			w.Fail(common.InternalServerError, "")
			return
		}

		// Memoize parent namespace
		parent = ns

		// Scan roles for permissions
		for _, role := range roles {
			if ns.HasPermission(role, createStatement.RequiredPermissions()) {
				access = true
			}
		}

		// Return error if not authorized
		if !access {
			w.Fail(common.Unauthorized, "cannot create namespace '%s'", namespace)
			return
		}
	}

	// If we've gotten this far, the user has permission to create the namespace

	// Get parent namespace
	if parent == nil {

		// Verify namespace existance
		parent, err = namespaceStore.Get(parentNamespace)
		if err != nil {
			w.Fail(common.InternalServerError, "parent namespace does not exist")
			return
		}
	}

	// Create child namespace
	if _, err = parent.CreateChild(namespace); err != nil {
		w.Fail(common.CreateNamespaceError, "cannot create namespace '%s'", namespace)
		return
	}

	w.Success(common.OK, "namespace created")
}

func (e *Executor) handleShowNamespace(w *common.ResponseWriter, stmt skl.Statement) {

	_, ok := stmt.(*skl.ShowNamespacesStatement)
	if !ok {
		w.Fail(common.InvalidStatementType, "expected *ShowNamespacesStatement, got %s instead", reflect.TypeOf(stmt))
		return
	}

	// Get session user
	user := e.session.user
	if user.IsAdmin() {

		// Get namespace store
		namespaceStore, err := e.system.Namespaces()
		if err != nil {
			w.Fail(common.InternalServerError, "could not access namespace data")
			return
		}

		// Stream namespaces
		w.Write(w.Colors.LightYellow)
		namespaces := namespaceStore.Stream()
		for name := range namespaces {
			w.Write([]byte(" " + name + "\r\n"))
		}
		w.Write(w.Colors.Reset)
	} else {

		// List namespaces
		w.Write(w.Colors.Yellow)
		for _, name := range user.Namespaces() {
			w.Write([]byte(" " + name + "\r\n"))
		}
		w.Write(w.Colors.Reset)
	}

	w.Success(common.OK, "")
}

// namespaceAlreadyExists determines if a namespace already exists...
func (e *Executor) namespaceAlreadyExists(namespace string, store datamodel.NamespaceStore) bool {
	_, err := store.Get(namespace)
	return err == nil
}

// If the namespace being created is a root namespace, only the admin account can create it
func (e *Executor) handleCreateRootNamespace(w *common.ResponseWriter, stmt *skl.CreateNamespaceStatement, store datamodel.NamespaceStore) {

	// Get namespace
	name := stmt.Namespace()

	// Verify namespace existance
	_, err := store.Get(name)

	// If err == nil, the namespace already exists
	if err == nil {
		w.Success(common.NamespaceAlreadyExists, name)
		return
	}

	// Get session user
	user := e.session.user
	if user.IsAdmin() {

		// Create new namespace
		_, err := store.Create(name)

		// If err !+ nil, namespace could not be created
		if err != nil {
			w.Fail(common.CreateNamespaceError, "could not create namespace '%s'", name)
			return
		}

		// No error == success
		w.Success(common.OK, "namespace created")
		return
	}

	// Otherwise fail creation
	w.Fail(common.Unauthorized, "root namespaces can only be created by the admin account")
	return
}
