package proto

// These types make public the oneof types so that they can be passed as arguments.
// IMHO, these types should have been public already.
// This is not auto-generated and will have to be repaired if the oneof types names are changed.

type GameActionRequest_Type interface {
	isGameActionRequest_Type
}

type GameActivity_Type interface {
	isGameActivity_Type
}

type RegistryActivity_Type interface {
	isRegistryActivity_Type
}
