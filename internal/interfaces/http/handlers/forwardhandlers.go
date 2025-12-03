package handlers

import (
	forwardHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward"
)

type ForwardHandler = forwardHandlers.ForwardHandler

var NewForwardHandler = forwardHandlers.NewForwardHandler
