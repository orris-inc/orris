package handlers

import (
	forwardHandlers "orris/internal/interfaces/http/handlers/forward"
)

type ForwardHandler = forwardHandlers.ForwardHandler

var NewForwardHandler = forwardHandlers.NewForwardHandler
