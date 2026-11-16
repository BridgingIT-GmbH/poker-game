SRC_DIR := src/main/java
OUT_DIR := out
MAIN_CLASS := pokergame.Main
MAIN_SRC := $(SRC_DIR)/pokergame/Main.java

.PHONY: build
build:
	@mkdir -pv $(OUT_DIR)
	javac -d $(OUT_DIR) -sourcepath $(SRC_DIR) $(MAIN_SRC)

.PHONY: run
run: build
	java -cp $(OUT_DIR) $(MAIN_CLASS) $(ARGS)

.PHONY: clean
clean:
	rm -rf $(OUT_DIR)

