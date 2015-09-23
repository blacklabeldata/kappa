package serf

import (
	"github.com/hashicorp/serf/serf"

	"github.com/stretchr/testify/mock"
)

type MockMemberEventHandler struct {
	mock.Mock
}

func (m *MockMemberEventHandler) HandleMemberEvent(e serf.MemberEvent) {
	m.Called(e)
	return
}

type MockUserEventHandler struct {
	mock.Mock
}

func (m *MockUserEventHandler) HandleUserEvent(e serf.UserEvent) {
	m.Called(e)
	return
}

type MockReconciler struct {
	mock.Mock
}

func (m *MockReconciler) Reconcile(e serf.MemberEvent) {
	m.Called(e)
	return
}
