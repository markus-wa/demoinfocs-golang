package sendtables2

func readFields(r *reader, s *serializer, state *fieldState) {
	fps := readFieldPaths(r)

	for _, fp := range fps {
		decoder := s.getDecoderForFieldPath(fp, 0)

		val := decoder(r)
		state.set(fp, val)

		fp.release()
	}
}
